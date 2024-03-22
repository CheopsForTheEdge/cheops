# A routine to wait for hosts to be synchronized

import time
import requests

def wait(hosts):
    def is_synchronized():
        for host in hosts:
            sched = requests.get(f"http://{host}:5984/_scheduler/docs", auth=("admin", "password"))
            for doc in sched.json()['docs']:
                if 'info' in doc and doc['info'] and 'changes_pending' in doc['info']:
                    if doc['info']['changes_pending'] and doc['info']['changes_pending'] > 0:
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
