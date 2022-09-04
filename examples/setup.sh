#!/usr/bin/env bash
# Copyright 2022 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

ip netns add ns1
ip netns add ns2

ip link add veth-ns1 type veth peer name veth-ns2
ip link set veth-ns1 netns ns1
ip link set veth-ns2 netns ns2
ip netns exec ns1 ip a add 10.0.0.1/24 dev veth-ns1
ip netns exec ns2 ip a add 10.0.0.2/24 dev veth-ns2
ip netns exec ns1 ip link set veth-ns1 up
ip netns exec ns2 ip link set veth-ns2 up

