package fr.inria.stack.cheops.base;

import fr.inria.stack.cheops.util.Conversion;

import io.etcd.jetcd.ByteSequence;
import io.etcd.jetcd.Client;
import io.etcd.jetcd.KV;
import io.etcd.jetcd.KeyValue;
import io.etcd.jetcd.kv.GetResponse;

import java.io.*;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.ExecutionException;
/*
* The etcd database abstraction layer
* */

public class ObjectsStoreEtcd {
    private String endpoint;
    private Client etcdClient;
    private KV kvClient;

    public ObjectsStoreEtcd(String endpoint) {
        this.endpoint = endpoint;
    }

    public void establishClient()  {
        etcdClient = Client.builder().endpoints(endpoint).build();
        kvClient = etcdClient.getKVClient();
    }

    /*
    * This function blocks and returns only a single object
    * You must call it iff the key always returns
    * a single object.
    * example: /resources/vms/uid, we know
    * this key always returns a single VM object
    * */
    public Object getFirstObjectSync(String key) throws ExecutionException, InterruptedException, IOException, ClassNotFoundException {
        ByteSequence bkey = ByteSequence.from(key.getBytes());

        CompletableFuture<GetResponse> getFuture = kvClient.get(bkey);
        GetResponse response = getFuture.get();

        ByteSequence value = response.getKvs().get(0).getValue();
        return Conversion.convertByteSequenceToObject(value);
    }

    public void putObject(String key, Serializable object) throws IOException, ExecutionException, InterruptedException {
        ByteSequence bkey = ByteSequence.from(key.getBytes());
        ByteSequence bvalue = Conversion.convertObjectToByteSequence(object);

        kvClient.put(bkey, bvalue).get();
    }
}
