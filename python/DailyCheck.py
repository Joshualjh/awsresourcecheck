import boto3
import urllib3
import json
from threading import Thread
http = urllib3.PoolManager()

def Ec2StatusCheck(): # ec2 status check
    ec2 = boto3.resource('ec2')
    for instance in ec2.instances.all():
        if instance.state['Name'] != 'running':
            SendEc2Status(instance.id, instance.state['Name'])

def SendEc2Status(id,state): ## ec2 status webhook
    url = ""
    encoded_msg =  json.dumps(
        {
            "@type": "MessageCard",
            "@context": "http://schema.org/extensions",
            "themeColor": "0076D7", 
            "summary": "AWS status check",
            "sections": [{
                "activityTitle": "AWS EC2 Daily check",
                "activitySubtitle": "your instance is not running",
                "activityImage": "https://tipsformsp.atlassian.net/60eb2e4d-8d54-4da3-bcb7-c8829b44e7b9#media-blob-url=true&id=68a18d6c-3c0c-42ad-aa80-a3a3b1fe69f3&contextId=33040&collection=contentId-33040",
                "facts": [{
                    "name": "instance-id",
                    "value": id,
                }, 
                {
                    "name": "Status",
                    "value": state,
                }],
            }],
        }
        )
    response = http.request('POST', url,headers={'Content-Type': 'application/json'} ,body=encoded_msg)

def KeyPairCheck(): ## keypair check
    ec2 = boto3.client('ec2')
    response = ec2.describe_key_pairs()
    data = []

    for i in response['KeyPairs']:
        count = 0
        for w in range(len(data)):
            if i['KeyName'] == data[w]:
                count += 1
        if count != 1:
            SendKeyPair(i["KeyName"], len(response['KeyPairs']))
    
def SendKeyPair(keyname, sumkey): ## keypair webhook
    url = ""
    encoded_msg =  json.dumps(
        {
            "@type": "MessageCard",
            "@context": "http://schema.org/extensions",
            "themeColor": "0076D7", 
            "summary": "AWS status check",
            "sections": [{
                "activityTitle": "AWS Key Daily check",
                "activitySubtitle": "there is new key",
                "activityImage": "https://tipsformsp.atlassian.net/60eb2e4d-8d54-4da3-bcb7-c8829b44e7b9#media-blob-url=true&id=68a18d6c-3c0c-42ad-aa80-a3a3b1fe69f3&contextId=33040&collection=contentId-33040",
                "facts": [{
                    "name": "New key",
                    "value": keyname,
                },
                {
                    "name": "number of keys",
                    "value": sumkey,
                },
                ],
            }],
        }
        )
    response = http.request('POST', url,headers={'Content-Type': 'application/json'} ,body=encoded_msg)

def EOF(): ## daily check last webhook
    url = ""
    encoded_msg =  json.dumps(
        {
            "@type": "MessageCard",
            "@context": "http://schema.org/extensions",
            "themeColor": "0076D7", 
            "summary": "last",
            "sections": [{
                "activityTitle": "Daily check",
                "activitySubtitle": "your instance is not running",
                "activityImage": "https://tipsformsp.atlassian.net/60eb2e4d-8d54-4da3-bcb7-c8829b44e7b9#media-blob-url=true&id=68a18d6c-3c0c-42ad-aa80-a3a3b1fe69f3&contextId=33040&collection=contentId-33040",
                "facts": [{
                    "name": "Daily check",
                    "value": "DONE",
                }, 
                ],
            }],
        }
        )
    response = http.request('POST', url,headers={'Content-Type': 'application/json'} ,body=encoded_msg)

if __name__ == "__main__":
    th1 = Thread(target=Ec2StatusCheck, args=())
    th2 = Thread(target=KeyPairCheck, args=())
    th1.start() ## ec2 status thread start
    th2.start() ## keyPair status thread start
    th1.join()  ## wait ec2 status thread end 
    th2.join()  ## wait status thread end
    EOF()       ## if all thread finish then go to EOF



