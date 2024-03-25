# A routine to wait for hosts to be synchronized

import time
import requests

def wait(hosts):
    def is_synchronized():
        for host in hosts:
            # Synchronization of documents: if not all documents are everywhere, we're not done yet
            try:
                sched = requests.get(f"http://{host}:5984/_scheduler/docs", auth=("admin", "password"), timeout=1)
            except requests.exceptions.ConnectTimeout:
                return False
            for doc in sched.json()['docs']:
                if 'info' in doc and doc['info'] and 'changes_pending' in doc['info']:
                    if doc['info']['changes_pending'] and doc['info']['changes_pending'] > 0:
                        return False

            # Synchronization of replies: now that all documents are everywhere, check that all operations
            # are run the same everywhere
            replies = requests.get(f"http://{host}:5984/cheops/_design/cheops/_view/last-reply", params={"group_level": 2}, timeout=1)
            # For each resourceid, gather by id then by requestid, and count unique requestids
            a = {}
            for row in replies.json()['rows']:
                id = row['key'][0]
                if id not in a:
                    a[id] = {}
                requestid = row['value']['RequestId']
                if requestid not in a[id]:
                    a[id][requestid] = 0
                a[id][requestid] += 1
            for byid in a.values():
                if len(byid) != 1:
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
