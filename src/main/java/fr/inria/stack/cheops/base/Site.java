package fr.inria.stack.cheops.base;

import fr.inria.stack.cheops.util.UID;

import java.io.Serializable;

public class Site {
    private String uid;
    private String name;
    private String region;
    private String api;

    public Site() {
        this.uid = UID.generateRandomUID();
    }
    public Site(String name, String region, String api) {
        this.uid = UID.generateRandomUID();
        this.name = name;
        this.region = region;
        this.api = api;
    }

    public String getUid() {
        return uid;
    }

    public void setUid(String uid) {
        this.uid = uid;
    }

    public String getName() {
        return name;
    }

    public void setName(String name) {
        this.name = name;
    }

    public String getRegion() {
        return region;
    }

    public void setRegion(String region) {
        this.region = region;
    }

    public String getApi() {
        return api;
    }

    public void setApi(String api) {
        this.api = api;
    }
}
