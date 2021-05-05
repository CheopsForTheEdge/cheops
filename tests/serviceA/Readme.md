## General

Works on run (either with an IDE such as Pycharm or with the python command) or in a docker container.

From command:
```bash
virtualenv venv
source venv/bin/activate
pip install -r requirements.txt
python3 app.py
```

For docker, standalone (resourceafromb does not work then):
```bash
sudo docker build --tag servicea .
sudo docker run --name servicea -p 5001:5001 servicea # with or without -d after run (for the daemon)
```

For docker, with service b:
```bash
sudo docker build --tag servicea .
cd ../serviceB
sudo docker build --tag serviceb .
cd -
# WARNING: "app1" is essential as it is the name used by resourceafromb from serviceA to find serviceB (cf [app.py](../app.py#52) )
docker network create app1
sudo docker run -d --net app1 --name servicea -p 5001:5001 servicea
sudo docker run -d --net app1 --name serviceb -p 5002:5002 serviceb
```


## API

- POST /resourcea
  
  Creates a resourcea given a JSON.

  **Request:** A string *resource* in body, as JSON.
  
  **Response:** A JSON of the created resourcea, with an integer *id* and the string *resource*.
- POST /resourceafromb/{mode}
  
  Creates a resourcea given a resourceb[1].

  **Request:** A string *mode*, either run or docker, respectively if run without or with docker.
  
  **Response:** A JSON of the created resourcea, with an integer *id* and the string *resource*.
- GET /resourcea/{resourcea_id}
  
  Shows details of a resourcea, given an id.

  **Request:** The string *id* of object in the URL.
  
  **Response:** A JSON of the resourcea, with an integer *id* and the string *resource*.
- PUT /resourcea/{resourcea_id}
  
  Updates a resourcea given a json.

  **Request:** A string *resource* in body, as JSON, and a string *id* in the URL.
  
  **Response:** A JSON of the updated resourcea, with an integer *id* and the string *resource*.
- DELETE /resourcea/{resourcea_id}
  
  Deletes a resourcea given an id.

  **Request:** The string *id* of the object in the URL.
  
  **Response:** A JSON `{'success': True}`.



## Example

1. Create a resource
```
curl -X POST http://0.0.0.0:5001/resourcea -d '{"resource":"lol"}' -H "Content-Type: application/json"
```
{"id":1,"resource":"lol"}

---

2. Get the resource
```
curl -X GET http://0.0.0.0:5001/resourcea/1
```
{"id":1,"resource":"lol"}

---

3. Update the resource
```
curl -X PUT http://0.0.0.0:5001/resourcea/1 -d '{"resource":"lil"}' -H "Content-Type: application/json"
```
{"id":1,"resource":"lil"}

---

4. Delete the resource
```
curl -X DELETE http://0.0.0.0:5001/resourcea/1
```
{"success": true}

---

5. Create a resourcea from a resourceb
```
curl -X POST http://0.0.0.0:5001/resourceafromb/run
```
{
  "id": 1, 
  "resource": "Resource b"
}

## Thanks to these resources:
https://pythonbasics.org/flask-rest-api/

https://stackoverflow.com/questions/4315111/how-to-do-http-request-call-with-json-payload-from-command-line

https://www.geeksforgeeks.org/dockerize-your-flask-app/

https://www.datacamp.com/community/tutorials/making-http-requests-in-python

https://stackoverflow.com/questions/45481943/connecting-two-docker-containers

https://flask.palletsprojects.com/en/1.1.x/api/?highlight=route#url-route-registrations


[1] For now, works with Service b default resource