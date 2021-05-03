from flask import Flask
from flask import request
import json

from database.database import db, init_database
from database.models import ResourceA

app = Flask(__name__)
app.config["SECRET_KEY"] = "good_pwd"
app.config["SQLALCHEMY_DATABASE_URI"] = "sqlite:///database/database.db"
app.config["SQLALCHEMY_TRACK_MODIFICATIONS"] = False

db.init_app(app) # (1) flask prend en compte la base de donnee

with app.test_request_context(): # (2) bloc execute a l'initialisation de Flask
    init_database()


def save_object_to_db(db_object):
    db.session.add(db_object)
    db.session.commit()  # Sauvegarde les informations dans la base de donnees


def remove_object_from_db(db_object):
    db.session.delete(db_object)
    db.session.commit()


def find_resource_by_id(rsc_id):
    return ResourceA.query.filter_by(id=rsc_id).first()


@app.route("/resourcea", methods=["POST"])
def create_resource_a():
    rq = request.json
    print(rq)
    print(rq['resource'])
    rsc_a = ResourceA(resource=rq['resource'])
    save_object_to_db(rsc_a)
    return rsc_a.serialize()

@app.route("/resourcea/<int:resource_id>", methods=["GET", "PUT", "DELETE"])
def modify_resource_a(resource_id):
    if request.method == 'GET':
        rsc = find_resource_by_id(resource_id)
        if rsc is not None:
            return rsc.serialize()
        else:
            return 'Resource not found', 404
    if request.method == 'PUT':
        rsc_a_json = json.loads(request.data)
        rsc = find_resource_by_id(resource_id)
        if rsc is not None:
            rsc.resource = (rsc_a_json['resource'])
            save_object_to_db(rsc)
            return rsc.serialize()
        else:
            return 'Resource not found', 404
    if request.method == 'DELETE':
        rsc = find_resource_by_id(resource_id)
        if rsc is not None:
            remove_object_from_db(rsc)
            return json.dumps({'success': True}), 200, {'ContentType': 'application/json'}
        else:
            return 'Resource not found', 404

if __name__ == '__main__':
    app.run()
