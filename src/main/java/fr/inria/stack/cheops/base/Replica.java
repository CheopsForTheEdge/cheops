package fr.inria.stack.cheops.base;

public class Replica {
    private String guid;
    private String luid;    /* local UID */

    public String getGuid() {
        return guid;
    }

    public void setGuid(String guid) {
        this.guid = guid;
    }

    public String getLuid() {
        return luid;
    }

    public void setLuid(String luid) {
        this.luid = luid;
    }
}
