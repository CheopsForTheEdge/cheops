package fr.inria;

public abstract class AbstractResource implements IResource {

//    Create the resource in the database
    abstract public int create();

//    Update the resource given an id
    abstract public void update(int id);

//    Mark the resource as deleted for future delete
    abstract public void markAsDeleted(int id);

//    Delete the resource
    abstract public void delete(int id);

//    Get the resource given an id
    abstract public Resource getResource(int id);
}
