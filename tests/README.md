# Execute API tests

This part is to test the endpoints (from main.go). 

I used pyresttest to allow REST requests easily and tried it with testingAPI.
yaml with fake classes, didn't try with our classes except for get all 
replicants, which doesn't work at that time...

## To install pyresttest:

Run:
```
sudo apt install libssl-dev python-pycurl libcurl4-openssl-dev
virtualenv venv
source venv/bin/activate
pip install -r requirements.txt
```

## To execute the API tests:

Use:
```
resttest.py http://localhost:8080 testingAPI.yaml
```
