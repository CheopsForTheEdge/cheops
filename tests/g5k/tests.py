import g5k
import unittest
import requests
import enoslib as en

import firewall_block

class CheopsTest(unittest.TestCase):
    def do(self, id, index, request):
        r = requests.post(f"http://{g5k.hosts[index]}:8079/exec/{id}", files=request)
        self.assertEqual(200, r.status_code, f"{id}: {r.text}")

    def wait_and_verify(self, id, hosts=g5k.hosts[:3]):
        firewall_block.wait(hosts)

        replies = [requests.get(f"http://{host}:5984/cheops/{id}") for host in hosts]
        for reply in replies:
            self.assertEqual(200, reply.status_code)
            self.assertEqual(replies[0].json(), reply.json())
        contents = [reply.json() for reply in replies]
        for content in contents:
            self.assertEqual(content['Operations'], contents[0]['Operations'])

        # Make sure the replies are all ok
        for host in hosts:
            query = {"selector": {
                "Type": "REPLY",
                "Site": host,
                "ResourceId": id
            }}
            r = requests.post(f"http://{host}:5984/cheops/_find", json=query, headers={"Content-Type": "application/json"})
            for doc in r.json()['docs']:
                self.assertEqual("OK", doc['Status'], f"status is KO {doc}")

    def verify_shell(self, command):
        # Make sure the directory has the correct content everywhere
        with en.actions(roles=g5k.roles_for_hosts) as p:
            p.shell(command)
            results = p.results

        contents = [content.payload['stdout'] for content in results.filter(task="shell")]
        for content in contents[1:]:
            self.assertEqual(contents[0], content)


