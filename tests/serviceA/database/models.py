from database.database import db


class ResourceA(db.Model):
    id = db.Column(db.Integer, primary_key=True)
    resource = db.Column(db.Text)

    def serialize(self):
        return {
            'id': self.id,
            'resource': self.resource
        }

    def __repr__(self):
        return "{'id': %s, 'resource': %s}" % (self.id, self.resource)

