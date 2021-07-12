## General

ServiceB has a default resource created (cf [app.py](../app.py#52)) to be used by serviceA.


Works on run (either with an IDE such as Pycharm or with the python command) or in a docker container.

From command:
```bash
virtualenv venv
source venv/bin/activate
pip install -r requirements.txt
python3 app.py
```

For docker, standalone:
```bash
sudo docker build --tag serviceb .
sudo docker run --name serviceb -p 5002:5002 serviceb # with or without -d after run (for the daemon)
```

For docker, with service a:
```bash
sudo docker build --tag serviceb .
cd ../serviceA
sudo docker build --tag servicea .
cd -
# WARNING: "app1" is essential as it is the name used by resourceafromb from serviceA to find serviceB
docker network create app1
sudo docker run -d --net app1 --name servicea -p 5001:5001 servicea
sudo docker run -d --net app1 --name serviceb -p 5002:5002 serviceb
```

## API

- POST /resourceb
  
  Creates a resourceb given a JSON.

  **Request:** A string *resource* in body, as JSON.
  
  **Response:** A JSON of the created resourceb, with an integer *id* and the string *resource*.
- GET /resourceb/{resourceb_id}
  
  Shows details of a resourceb, given an id.

  **Request:** The string *id* of object in the URL.
  
  **Response:** A JSON of the resourceb, with an integer *id* and the string *resource*.
- PUT /resourceb/{resourceb_id}
  
  Updates a resourceb given a json.

  **Request:** A string *resource* in body, as JSON, and a string *id* in the URL.
  
  **Response:** A JSON of the updated resourceb, with an integer *id* and the string *resource*.
- DELETE /resourceb/{resourceb_id}
  
  Deletes a resourceb given an id.

  **Request:** The string *id* of the object in the URL.
  
  **Response:** A JSON `{'success': True}`.



## Example

1. Create a resource
```
curl -X POST http://0.0.0.0:5002/resourceb -d '{"resource":"lol"}' -H "Content-Type: application/json"
```
{"id":1,"resource":"lol"}

---

2. Get the resource
```
curl -X GET http://0.0.0.0:5002/resourceb/1
```
{"id":1,"resource":"lol"}

---

3. Update the resource
```
curl -X PUT http://0.0.0.0:5002/resourceb/1 -d '{"resource":"lil"}' -H "Content-Type: application/json"
```
{"id":1,"resource":"lil"}

---

4. Delete the resource
```
curl -X DELETE http://0.0.0.0:5002/resourceb/1
```
{"success": true}