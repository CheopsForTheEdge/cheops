import sys
import os
import socket
import enoslib as en

if any(['g5k-jupyterlab' in path for path in sys.path]):
    print("Running on Grid'5000 notebooks, applying workaround for https://intranet.grid5000.fr/bugzilla/show_bug.cgi?id=13606")
    print("Before:", sys.path)
    sys.path.insert(1, os.environ['HOME'] + '/.local/lib/python3.9/site-packages')
    print("After:", sys.path)

site = "nantes"
cluster = "ecotype"

en.init_logging()

network = en.G5kNetworkConf(type="prod", roles=["my_network"], site=site)

conf = (
    en.G5kConf.from_settings(job_type=[], walltime="01:50:00", job_name="cheops")
    .add_network_conf(network)
    .add_machine(
        roles=["cheops"],
        cluster=cluster,
        nodes=6,
        primary_network=network,
    )
    .finalize()
)

provider = en.G5k(conf)

rroles, networks = provider.init()
roles = rroles['cheops']

if len(roles) == 0:
   sys.exit("Didn't find roles")

en.sync_info(roles, networks)
hosts = [role.alias for role in roles]
sites = '&'.join(hosts)
roles_for_hosts = [role for role in roles if role.alias in hosts[:3]]
