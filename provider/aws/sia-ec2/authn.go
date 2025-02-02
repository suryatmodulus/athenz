//
// Copyright Athenz Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package sia

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/AthenZ/athenz/libs/go/sia/aws/meta"
	"github.com/AthenZ/athenz/libs/go/sia/aws/options"
	"github.com/AthenZ/athenz/libs/go/sia/util"
)

func getDocValue(docMap map[string]interface{}, key string) string {
	value := docMap[key]
	if value == nil {
		return ""
	} else {
		return value.(string)
	}
}

func GetEC2DocumentDetails(metaEndPoint string) ([]byte, []byte, string, string, string, *time.Time, error) {
	document, err := meta.GetData(metaEndPoint, "/latest/dynamic/instance-identity/document")
	if err != nil {
		return nil, nil, "", "", "", nil, err
	}
	signature, err := meta.GetData(metaEndPoint, "/latest/dynamic/instance-identity/pkcs7")
	if err != nil {
		return nil, nil, "", "", "", nil, err
	}
	var docMap map[string]interface{}
	err = json.Unmarshal(document, &docMap)
	if err != nil {
		return nil, nil, "", "", "", nil, err
	}
	account := getDocValue(docMap, "accountId")
	region := getDocValue(docMap, "region")
	instanceId := getDocValue(docMap, "instanceId")

	timeCheck, _ := time.Parse(time.RFC3339, getDocValue(docMap, "pendingTime"))
	return document, signature, account, instanceId, region, &timeCheck, err
}

func GetECSOnEC2TaskId() string {
	ecs := os.Getenv("ECS_CONTAINER_METADATA_FILE")
	if ecs == "" {
		log.Println("Not ECS on EC2 instance")
		return ""
	}
	ecsMetaData, err := ioutil.ReadFile(ecs)
	if err != nil {
		log.Printf("Unable to read ECS on EC2 instance metadata: %s - %v\n", ecs, err)
		return ""
	}
	var docMap map[string]interface{}
	err = json.Unmarshal(ecsMetaData, &docMap)
	if err != nil {
		log.Printf("Unable to parse ECS on EC2 instance metadata: %s - %v\n", ecs, err)
		return ""
	}
	taskArn := getDocValue(docMap, "TaskARN")
	_, taskId, _, err := util.ParseTaskArn(taskArn)
	if err != nil {
		log.Printf("Unable to parse ECS on EC2 task id: %s - %v\n", taskArn, err)
		return ""
	}
	return taskId
}

func GetEC2Config(configFile, metaEndpoint string, useRegionalSTS bool, region, account string) (*options.Config, *options.ConfigAccount, error) {
	config, configAccount, err := options.InitFileConfig(configFile, metaEndpoint, useRegionalSTS, region, account)
	if err != nil {
		log.Printf("Unable to process configuration file '%s': %v\n", configFile, err)
		log.Println("Trying to determine service details from the environment variables...")
		config, configAccount, err = options.InitEnvConfig(config)
		if err != nil {
			log.Printf("Unable to process environment settings: %v\n", err)
			// if we do not have settings in our environment, we're going
			// to use fallback to <domain>.<service>-service naming structure
			log.Println("Trying to determine service name security credentials...")
			configAccount, err = options.InitCredsConfig("-service", useRegionalSTS, region)
			if err != nil {
				log.Printf("Unable to process security credentials: %v\n", err)
				log.Println("Trying to determine service name from profile arn...")
				configAccount, err = options.InitProfileConfig(metaEndpoint, "-service")
				if err != nil {
					log.Printf("Unable to determine service name: %v\n", err)
					return config, nil, err
				}
			}
		}
	}
	return config, configAccount, nil
}
