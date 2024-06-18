# A routine to wait for hosts to be synchronized

import time
import requests
import enoslib as en

def activate(roles):
    with en.actions(roles=roles) as p:
        p.iptables(
                chain="INPUT",
                source="127.0.0.1",
                jump="ACCEPT",
                state="present"
        )
        p.iptables(
                chain="INPUT",
                protocol="tcp",
                destination_port="5984",
                jump="DROP",
                state="present"
        )

    # Wait for blocking to be in place
    import time
    time.sleep(3)

def deactivate(roles):
    with en.actions(roles=roles) as p:
        p.iptables(
                chain="INPUT",
                source="127.0.0.1",
                jump="ACCEPT",
                state="absent"
        )
        p.iptables(
                chain="INPUT",
                protocol="tcp",
                destination_port="5984",
                jump="DROP",
                state="absent"
        )


    def is_synchronized():
        hosts = [r.alias for r in roles]
        for host in hosts:
            # Synchronization of documents: if not all documents are everywhere, we're not done yet
            try:
                sched = requests.get(f"http://{host}:5984/_scheduler/docs", auth=("admin", "password"), timeout=1)
            except requests.exceptions.ConnectTimeout:
                return False
            for doc in sched.json()['docs']:
                if 'info' in doc and doc['info'] and 'changes_pending' in doc['info']:
                    if doc['info']['changes_pending'] is None:
                        return False
                    if doc['info']['changes_pending'] and doc['info']['changes_pending'] > 0:
                        return False

            # Synchronization of resources
            res = requests.post(f"http://{host}:5984/cheops/_find", json={"selector": {"Type": "RESOURCE"}})
            for doc in res.json()["docs"]:
                if "_conflicts" in doc and len(doc["_conflicts"]) > 0:
                    return False

            # All operations are run
            for doc in res.json()["docs"]:
                resrep = requests.post(f"http://{host}:5984/cheops/_find", json={"selector": {"Type": "REPLY", "Site": host, "ResourceId": doc["_id"]}})
                ops = [op["RequestId"] for op in doc["Operations"]]
                resids = [rep["RequestId"] for rep in resrep.json()["docs"]]
                for op in ops:
                    if op not in resids:
                        return False

        return True

    while True:
        if is_synchronized():
            break
        else:
            time.sleep(1)

if __name__ == "__main__":
    import os
    path = os.path.expanduser("~/.oarnodes")
    with open(path) as nodes:
        hosts = [host.strip() for host in nodes.readlines()]
        wait(hosts)
