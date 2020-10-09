package fr.inria;

// Underlying implementation is the application that will execute the request (Kubernetes, OS, etc.)
public abstract class AbstractDriver {

//    Translate the request for the underlying implementation
    public abstract void translate();

//    Give the request to the underlying implementation
    public abstract void transfer();

//    Get the response from underlying implementation and interprets it for our core
    public abstract void interpret();
}
