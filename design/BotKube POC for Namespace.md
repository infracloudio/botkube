# To Achieve 
- limiting channel access to a specific namespace (only updated from specific namespace will be sent on the channel)
- limiting access to selected resources with limited operations from a specific channel (limiting what command are allowed from the channel)
  with having one channel with all privileges 

# Existing solution 
(To accept input configuration from the user)
In `comm_config.yaml` under slack, mattermost communication options define a channel in the way below
```
channels:
- name: channel1
  namespaces:
  - ns1
  - ns2
  kubectl:
    Enabled: true
    commands:
      # method which are allowed
      verbs: ["get", "logs"]
      # resource configuration which is allowed
      resources: ["deployments", "pods"]
```

# Different way to implement the same solution 
 
The intention here is to make configuration simple to understand by end-user

As of now in the existing structure, we have 2 different files for configuration 
1. comm_config.yaml      ( For external communication-related data )
2. resource_config.yaml  ( Limiting of resource information that botkube should scrap + settings )

To add these new features too: 
 - limit what can be run from the channel
 - what notification should be sent to the channel 
All command execution access related information or access profiles can be defined under 3rd file (say) `access_conf.yml`

# Implementation

### Existing structure 
##### comm_config.yaml
``` 
# Channels configuration
communications:
  # Settings for Slack
  slack:
    Enabled: false
    channel: 'SLACK_CHANNEL'
    token: 'SLACK_API_TOKEN'
    notiftype: short  
```
### proposed stuucture 
key concept: comm_config.yml contains a mapping of specific profile instead of specific channels 
##### comm_config.yaml
```
  
# Channels configuration
communications:
  # Settings for Slack
  slack:
    Enabled: false
    token: 'SLACK_API_TOKEN'
    notiftype: short  
    accessbindings:
      - channel_name: developers
        # all profiles listed under access_conf.yml to limit access
        profile: DEVELOPMENT    
      - channel_name: production
        profile: PRODUCTION
      - channel_name: admin
        profile: BOTKUBE_ADMIN


```

##### access_conf.yml
```
profiles:
  # based on use-case like profile for development environment
  - name: 'DEVELOPMENT'     
    namespaces:
    - ns1
    - ns2
    kubectl:
      Enabled: true
      commands:
        # method which are allowed
        verbs: ["get", "logs"]
        # resource configuration which is allowed
        resources: ["Deployments", "Pods", "Services"]
        # enable notification about specific resources
           
   
  # Profile: For Production resources 
  - name: 'PRODUCTION'
    namespaces:
    - ns1
    - ns2
    kubectl:
      Enabled: true
      commands:
        # method which are allowed
        verbs: ["get", "logs", "describe", "diff", "explain", "top"]
        # resource configuration which is allowed
        resources: ["Deployments", "Pods", "Services", "Nodes", "Ingresses", "Roles"]  
        # enable notification about specific resources
          

  # Profile: BOTKUBE_ADMIN 
  - name: 'BOTKUBE_ADMIN'
    namespaces:
    # by default all namespaces will be included
    # - ns1
    # - ns2
    kubectl:
      Enabled: true
      commands:
        # method which are allowed
        verbs: ["get", "logs", "describe" ,"api-resources", "api-versions", "cluster-info", "diff", "explain", "top", "auth"]
        # resource configuration which is allowed
        resources: [ "Pods", "Services","Namespaces", "Nodes","ReplicationControllers", "PersistentVolumes", "PersistentVolumeClaims", "Secrets", "ConfigMaps", "Deployments", "DaemonSets", "ReplicaSets", "StatefulSets", "Ingresses", "Jobs", "Roles", "RoleBindings", "ClusterRoles"]  
        # enable notification about specific resources
```


# Benefits of extra file
- Profiles will be common for all communication options (eg: slack, mattermost and coming ones can utilize common profiles)
- More than one access profiles can be pre-defined and can be changed quickly when needed 
- This file will not be required for webhook and elastic search (need to discuss)
 
 
# Steps to implement 
  After SETUP (1st step below) all Parts can also be developed in parallel
    
==============
## SETUP
1. update the 'settings' block and move options to 'resource_config.yaml' (all this will move to new 'access_conf.yml')  
2. create new default 'access_conf.yml'
3. Updates in pkg "config/config.go" to reade and associate current 'profile' information under "type Config struct"

===============
## Part: 1 
   Focus on running kubectl commands sent by users corresponding to specific profiles
1. updates in Bot package to (For both slack and mattermost and optionally for elastic search and webhook)
    - initialize new configuration with the selected profile
    - HandleMessage function channel verification need to be updated as selected profile
2. execute package 
    - check for extra conditions (filters) as described under profile
    - check for Namespace(s) that channel has access 
3. limit access of @botkube notifier and @botkube filter command to channels under specific profiles     
4. update upgrade.go to send upgrade notification to channels under specific profiles 

===============
## Part: 2 
    Focus on message filter by informers to send the only right message on a specific channel
1.  Implement logic for every event base on profile looping over 
    corresponding channels and their profile-specific filter 
    to send an event to one or more channel as required should be implemented 
    in "sendEvent" function of controller file
    => https://github.com/infracloudio/botkube/blob/develop/pkg/controller/controller.go#L232

===============
## Part: 3 
 Update go Tests for check related to profiles 
 (need to plan)

===============
## Part: 4 (optional)
 Manage profile options using slack message itself something looks like  
 > @botkube profile (list, enable, update) that work only from any channel under botkube_admin profile

===============
