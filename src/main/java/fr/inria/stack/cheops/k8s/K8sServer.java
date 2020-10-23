package fr.inria.stack.cheops.k8s;

import java.io.IOException;
import java.io.Serializable;
import java.util.ArrayList;
import java.util.concurrent.ExecutionException;

import fr.inria.stack.cheops.base.*;

public class K8sServer {
    ObjectsStoreEtcd store;
    ArrayList<K8sSiteConnector> sites;

    public K8sServer(ObjectsStoreEtcd store) {
        this.store = store;
    }

    public void PutGetSiteTest() {
        Site site1 = new K8sSite("nantes", "bretagne", "https://nantes/api");
        Site site2 = new K8sSite("paris", "ile-de-france", "https://paris/api");

        try {
            store.putObject("/resource/site/site1", (Serializable) site1);
            System.out.println("Write has been performed");
            Site s = (Site) store.getFirstObjectSync("/resource/site/site1");
            System.out.println("name=" + s.getName() + " region=" + s.getRegion() + " api=" + s.getApi());
        } catch (Exception e) {
            e.printStackTrace();
            System.out.println(e.toString());
        }
    }
}
