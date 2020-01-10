NAME = load-balancer
DESCRIPTION = "A simple VPP load-balancer example"
CONTAINERS = pod
AUX_CONTAINERS =
PODS = simple-client load-balancer
NETWORK_SERVICES = load-balancer
CHECK = scripts/check_load_balancer.sh

include $(TOP)/mk/targets.mk
