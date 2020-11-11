# To Achieve 
- limiting channel access to a specific namespace (only updates - Kubectl commands from specific namespace will be available on the channel), also removing access for notifier start/stop, filter enable/disable
- limiting access to selected resources with limited operations from a specific channel (limiting what command are allowed from the channel)
  with having one channel with all privileges 
- notifications for events for all channels will be send as per the configuration defined in resource_config.yaml (To be discussed)
 
# Existing solution 
(To accept input configuration from the user)
In `comm_config.yaml` under slack, mattermost, discord, communication options define a channel in the way below
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
 
### Note 
- Microsoft Teams, ElasticSearch will fetch configurations from resource_config.yaml (settings -> kubectl)


## (optional)
 Manage profile options using slack message itself something looks like  
 > @botkube profile (list, enable, update) that work only from any channel under botkube_admin profile

===============