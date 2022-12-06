package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

var wg sync.WaitGroup
var stdin = bufio.NewReader(os.Stdin)

type EC2DescribeInstancesAPI interface {
	DescribeInstances(ctx context.Context,
		params *ec2.DescribeInstancesInput,
		optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
}

func GetInstances(c context.Context, api EC2DescribeInstancesAPI, input *ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error) {
	return api.DescribeInstances(c, input)
}

func userPolicyHasAdmin(user types.UserDetail, admin string) bool {
	for _, policy := range user.UserPolicyList {
		if *policy.PolicyName == admin {
			return true
		}
	}

	return false
}

func attachedUserPolicyHasAdmin(user types.UserDetail, admin string) bool {
	for _, policy := range user.AttachedManagedPolicies {
		if *policy.PolicyName == admin {
			return true
		}
	}

	return false
}

func groupPolicyHasAdmin(c context.Context, client *iam.Client, group types.Group, admin string) (bool, error) {
	input := &iam.ListGroupPoliciesInput{
		GroupName: group.GroupName,
	}

	result, err := client.ListGroupPolicies(c, input)
	if err != nil {
		return false, err
	}

	// Wade through policies
	for _, policyName := range result.PolicyNames {
		if policyName == admin {
			return true, nil
		}
	}

	return false, nil
}

func attachedGroupPolicyHasAdmin(c context.Context, client *iam.Client, group types.Group, admin string) (bool, error) {
	input := &iam.ListAttachedGroupPoliciesInput{
		GroupName: group.GroupName,
	}

	result, err := client.ListAttachedGroupPolicies(c, input)
	if err != nil {
		return false, err
	}

	for _, policy := range result.AttachedPolicies {
		if *policy.PolicyName == admin {
			return true, nil
		}
	}

	return false, nil
}

func usersGroupsHaveAdmin(c context.Context, client *iam.Client, user types.UserDetail, admin string) (bool, error) {
	input := &iam.ListGroupsForUserInput{
		UserName: user.UserName,
	}

	result, err := client.ListGroupsForUser(c, input)
	if err != nil {
		return false, err
	}
	//그룹
	//fmt.Println(result.Groups)

	for _, group := range result.Groups {
		groupPolicyHasAdmin, err := groupPolicyHasAdmin(c, client, group, admin)
		if err != nil {
			return false, err
		}

		if groupPolicyHasAdmin {
			return true, nil
		}

		attachedGroupPolicyHasAdmin, err := attachedGroupPolicyHasAdmin(c, client, group, admin)
		if err != nil {
			return false, err
		}

		if attachedGroupPolicyHasAdmin {
			return true, nil
		}
	}

	return false, nil
}

// GetNumUsersAndAdmins determines how many users have administrator privileges.
// Inputs:
//
//	client is the AWS Identity and Access Management (IAM) service client.
//	c is the context of the method call, which includes the AWS Region.
//
// Output:
//
//	If success, the list of users and admins, and nil.
//	Otherwise, "", "" and an error.
func GetNumUsersAndAdmins(c context.Context, client *iam.Client) (string, string, error) {
	users := ""
	admins := ""

	filters := make([]types.EntityType, 1)
	filters[0] = types.EntityTypeUser

	input := &iam.GetAccountAuthorizationDetailsInput{
		Filter: filters,
	}

	resp, err := client.GetAccountAuthorizationDetails(c, input)
	if err != nil {
		return "", "", err
	}
	fmt.Println(&resp.Policies)

	// The policy name that indicates administrator access
	adminName := "AdministratorAccess"

	// Wade through resulting users
	for _, user := range resp.UserDetailList {
		isAdmin, err := isUserAdmin(c, client, user, adminName)
		if err != nil {
			return "", "", err
		}

		users += " " + *user.UserName

		if isAdmin {
			admins += " " + *user.UserName
		}
	}

	for resp.IsTruncated {
		input := &iam.GetAccountAuthorizationDetailsInput{
			Filter: filters,
			Marker: resp.Marker,
		}

		resp, err = client.GetAccountAuthorizationDetails(c, input)
		if err != nil {
			return "", "", err
		}

		// Wade through resulting users
		for _, user := range resp.UserDetailList {
			isAdmin, err := isUserAdmin(c, client, user, adminName)
			if err != nil {
				return "", "", err
			}

			users += " " + *user.UserName

			if isAdmin {
				admins += " " + *user.UserName
			}
		}
	}

	return users, admins, nil
}

func isUserAdmin(c context.Context, client *iam.Client, user types.UserDetail, admin string) (bool, error) {
	// Check policy, attached policy, and groups (policy and attached policy)
	policyHasAdmin := userPolicyHasAdmin(user, admin)
	if policyHasAdmin {
		return true, nil
	}

	attachedPolicyHasAdmin := attachedUserPolicyHasAdmin(user, admin)
	if attachedPolicyHasAdmin {
		return true, nil
	}

	userGroupsHaveAdmin, err := usersGroupsHaveAdmin(c, client, user, admin)
	if err != nil {
		return false, err
	}
	if userGroupsHaveAdmin {
		return true, nil
	}

	return false, nil
}

func main() {
	// 로그인
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithSharedConfigProfile("test-account")) //test-account부분은 .aws/credencials참조
	// 로그인 실패시 에러 출력
	if err != nil {
		log.Fatalf("failed to load configuration, %v", err)
	}
	client := ec2.NewFromConfig(cfg)

	input := &ec2.DescribeInstancesInput{}
	//ec2 인스턴스 정보 가져오기
	result, err := GetInstances(context.TODO(), client, input)
	if err != nil {
		fmt.Println("Got an error retrieving information about your Amazon EC2 instances:")
		fmt.Println(err)
		return
	}

	for _, r := range result.Reservations {
		fmt.Println("Reservation ID: " + *r.ReservationId)
		fmt.Println("Instance IDs:")
		for _, i := range r.Instances {
			fmt.Println("   " + *i.InstanceId)
			fmt.Println(*i.State)
			for z := 0; z < len(i.SecurityGroups); z++ {
				fmt.Println(*&i.SecurityGroups[0])
			}

		}
		fmt.Println("")
	}

	client1 := iam.NewFromConfig(cfg)

	users, admins, err := GetNumUsersAndAdmins(context.TODO(), client1)
	if err != nil {
		fmt.Println("Got an error finding users who are admins:")
		fmt.Println(err)
		return
	}

	userList := strings.Split(users, " ")
	adminList := strings.Split(admins, " ")

	fmt.Println("")
	fmt.Println("Found", len(adminList)-1, "admin(s) out of", len(userList)-1, "user(s)")
	fmt.Println(adminList, "user account is", userList[1], ",", userList[2])
	// showDetails := flag.Bool("d", false, "Whether to print out names of users and admins")
	// if *showDetails {
	// 	fmt.Println("")
	// 	fmt.Println("Users")
	// 	for _, u := range userList {
	// 		fmt.Println("  " + u)
	// 	}

	// 	fmt.Println("")
	// 	fmt.Println("Admins")
	// 	for _, a := range adminList {
	// 		fmt.Println("  " + a)
	// 	}
	// }
}
