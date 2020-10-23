package fr.inria.stack.cheops.util;

import io.etcd.jetcd.ByteSequence;

import java.io.*;

public class Conversion {
    public static Object convertByteSequenceToObject(ByteSequence value) throws IOException, ClassNotFoundException {
        return new ObjectInputStream(
                new ByteArrayInputStream(
                        value.getBytes())).readObject();
    }

    public static ByteSequence convertObjectToByteSequence(Serializable object) throws IOException {
        ByteArrayOutputStream byteStream = new ByteArrayOutputStream();
        ObjectOutputStream objectStream = new ObjectOutputStream(byteStream);
        objectStream.writeObject(object);
        objectStream.flush();
        byteStream.close();
        return ByteSequence.from(byteStream.toByteArray());
    }
}
