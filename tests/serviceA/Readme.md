## API

- POST /resourcea
  
  Creates a resourcea given a JSON.

  **Request:** A string *resource* in body, as JSON.
  
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
curl -X POST http://127.0.0.1:5000/resourcea -d '{"resource":"lol"}' -H "Content-Type: application/json"
```
{"id":1,"resource":"lol"}

---

2. Get the resource
```
curl -X GET http://127.0.0.1:5000/resourcea/1
```
{"id":1,"resource":"lol"}

---

3. Update the resource
```
curl -X PUT http://127.0.0.1:5000/resourcea/1 -d '{"resource":"lil"}' -H "Content-Type: application/json"
```
{"id":1,"resource":"lil"}

---

4. Delete the resource
```
curl -X DELETE http://127.0.0.1:5000/resourcea/1
```
{"success": true}