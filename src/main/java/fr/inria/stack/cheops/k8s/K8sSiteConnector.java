package fr.inria.stack.cheops.k8s;

/* This is an in-memory structure that contains necessary
 * objects and references to establish connection with the
 * remote site's API
 */

import io.kubernetes.client.openapi.ApiClient;
import io.kubernetes.client.openapi.ApiException;
import io.kubernetes.client.openapi.apis.CoreV1Api;
import io.kubernetes.client.util.ClientBuilder;
import io.kubernetes.client.util.KubeConfig;

import java.io.FileReader;
import java.io.IOException;

public class K8sSiteConnector {
    private String uid;
    private String kubeConfigPath;
    private ApiClient client;
    private CoreV1Api corev1api;

    public K8sSiteConnector(String uid, String kubeConfigPath) {
        this.uid = uid;
        this.kubeConfigPath = kubeConfigPath;
    }

    public String getUid() {
        return uid;
    }

    public void setUid(String uid) {
        this.uid = uid;
    }

    public String getKubeConfigPath() {
        return kubeConfigPath;
    }

    public void setKubeConfigPath(String kubeConfigPath) {
        this.kubeConfigPath = kubeConfigPath;
    }

    public ApiClient getClient() {
        return client;
    }

    public void setClient(ApiClient client) {
        this.client = client;
    }

    public CoreV1Api getCorev1api() {
        return corev1api;
    }

    public void setCorev1api(CoreV1Api corev1api) {
        this.corev1api = corev1api;
    }

    public void establishAPIClient() throws IOException {
        ApiClient client =
                ClientBuilder.kubeconfig(
                        KubeConfig.loadKubeConfig(
                                new FileReader(kubeConfigPath))).build();
    }

    public Boolean checkSiteIsAlive() {
        if (client != null) {
            try {
                corev1api.listNamespace(null,
                        null,
                        null,
                        null,
                        null,
                        null,
                        null,
                        null,
                        null);
                return true;
            } catch (ApiException e) {
                return false;
            }
        }
        return false;
    }
}
