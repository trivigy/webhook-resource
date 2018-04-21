> It is without say that you need to ensure all the necessary tooling 
application like `gcloud`, `docker`, `kubectl`, `helm`, etc. are installed 
and configured.

### Build and push docker image
```bash
gcloud auth login

VERSION=latest
PROJECT=syncaide-200904
docker build -t gcr.io/${PROJECT}/revgcs:${VERSION} .
```

### Pull docker image
```bash
VERSION=latest
PROJECT=syncaide-200904
docker push gcr.io/${PROJECT}/revgcs:${VERSION}
```

### Configurations
```bash
gcloud auth application-default login
docker network create --driver=bridge --subnet=172.20.0.0/16 syncaide

PROJECT=syncaide-200904
docker run --rm -d -p 8080 \
    --network syncaide --ip 172.20.0.2 \
    -v ~/.config/gcloud:/root/.config/gcloud \
    gcr.io/${PROJECT}/revgcs:latest revgcs --bind 0.0.0.0:8080
    
helm repo add syncaide http://172.20.0.2:8080/static.syncaide.com/charts
```