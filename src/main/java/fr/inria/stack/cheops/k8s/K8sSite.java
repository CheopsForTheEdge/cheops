package fr.inria.stack.cheops.k8s;

import fr.inria.stack.cheops.base.*;

import java.io.Serializable;

public class K8sSite extends Site implements Serializable {
    public K8sSite(String name, String region, String api) {
        super(name, region, api);
    }
}

