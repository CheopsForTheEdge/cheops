package fr.inria.stack.cheops.base;

public class Resource {
    private String guid;
    private String type;

    public Resource(String guid, String type) {
        this.guid = guid;
        this.type = type;
    }

    public String getType() {
        return type;
    }

    public void setType(String type) {
        this.type = type;
    }

    public String getGuid() {
        return guid;
    }

    public void setGuid(String guid) {
        this.guid = guid;
    }
}
