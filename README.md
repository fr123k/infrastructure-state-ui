# Terraform UI

Shows the state of performed terraform plan from sre-openstack jenkins job.
[terraform dashboard]()

## Init 

```
go mod init terraform-ui
go mod vendor
```

## Run

Open the browser at localhost:8080 and it will display the ui with empty plans
```
make local
```
To upload a plan example just perform the following command
```
curl -m 30 -s -H 'Content-Type:application/json' -X POST -d @test/terraform.json http://localhost:8080/api/plan/test-project/workspace/default/version/master
```

## Changes

## Todo

* cleanup go code (one of the first go programs of myself)
* implement cleanup of plan files after a certain amount or age
* implement setting of plan metadata like project, version, ...
* apply fr123k brand with proper frontend design
* production deployment to a cloud provider most properly hetzner
