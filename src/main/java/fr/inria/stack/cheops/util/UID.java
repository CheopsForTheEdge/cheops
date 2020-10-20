package fr.inria.stack.cheops.util;

import java.util.UUID;

public class UID {
    /* 36 chars length UIDs
    *  example: 54947df8-0e9e-4471-a2f9-9af509fb5889
    */
    public static String generateRandomUID() {
        return UUID.randomUUID().toString();
    }
}
