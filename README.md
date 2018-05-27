> It is without say that you need to ensure all the necessary tooling 
application like `gcloud`, `docker`, `kubectl`, `helm`, etc. are installed 
and configured.

### Build and push docker image
```bash
gcloud auth login
docker build -t gcr.io/syncaide-200904/webhook-resource:latest .
```

### Pull docker image
```bash
docker push gcr.io/syncaide-200904/webhook-resource:latest
```