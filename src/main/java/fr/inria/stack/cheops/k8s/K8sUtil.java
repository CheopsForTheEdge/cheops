package fr.inria.stack.cheops.k8s;

import io.kubernetes.client.openapi.ApiClient;
import io.kubernetes.client.openapi.Configuration;

public class K8sUtil {
    public static void setClient(ApiClient client) {
        Configuration.setDefaultApiClient(client);
    }
}
