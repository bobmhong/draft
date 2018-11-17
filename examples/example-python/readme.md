#Stuff to do to make this work

to address warning about no registry set
```
draft config set registry docker.io/bobmhong
```

## Use Openshift Docker Daemon
eval $(minishift docker-env)

## Access Internal Registry
```
docker tag my-app $(minishift openshift registry)/myproject/my-app
```

## Do this if you don't want to create a route (for testing)
oc port-forward example-go-go-676f54c76d-tlvlh 8001:8080




export TILLER_NAMESPACE=tiller
oc policy add-role-to-user edit "system:serviceaccount:${TILLER_NAMESPACE}:tiller"

if draft up fails with helm cant upgrade error, then do helm delete example-python


