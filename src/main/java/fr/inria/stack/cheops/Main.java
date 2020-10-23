package fr.inria.stack.cheops;

import fr.inria.stack.cheops.base.ObjectsStoreEtcd;
import fr.inria.stack.cheops.k8s.K8sServer;

public class Main {
    public static void main(String[] args) {
        ObjectsStoreEtcd store = new ObjectsStoreEtcd("http://127.0.0.1:2379");
        K8sServer server = new K8sServer(store);
        store.establishClient();
        server.PutGetSiteTest();
    }
}
