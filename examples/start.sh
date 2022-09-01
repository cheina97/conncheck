#!/usr/bin/env bash

ip netns add ns1
ip netns add ns2

ip link add veth-ns1 type veth peer name veth-ns2
ip link set veth-ns1 netns ns1
ip link set veth-ns2 netns ns2
ip netns exec ns1 ip a add 10.0.0.1/24 dev veth-ns1
ip netns exec ns2 ip a add 10.0.0.2/24 dev veth-ns2
ip netns exec ns1 ip link set veth-ns1 up
ip netns exec ns2 ip link set veth-ns2 up
