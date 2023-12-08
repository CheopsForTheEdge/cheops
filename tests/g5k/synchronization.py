# A routine to wait for hosts to be synchronized

import time
import requests

def wait(hosts):
    def is_synchronized():
        for host in hosts:
            changes = requests.get(f"http://{host}:5984/cheops/_changes")
            current = changes.json()['last_seq']

            sched = requests.get(f"http://{host}:5984/_scheduler/docs", auth=("admin", "password"))
            for doc in sched.json()['docs']:
                if 'info' in doc and doc['info'] and 'source_seq' in doc['info'] and doc['info']['source_seq']:
                    synchronized = doc['info']['source_seq']
                    if synchronized != current:
                        return False
        return True

    while True:
        if is_synchronized():
            break
        else:
            time.sleep(1)
