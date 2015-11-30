# check_cisco_uc_perf
check_cisco_uc_perf is a Nagios plugin to monitor the performance of Cisco Unified Communications Servers


file: check_cisco_ucm_perf.go
Version 0.3.2 (27.02.2015)

check_cisco_ucm_perf is a Nagios plugin made by Herwig Grimm (herwig.grimm at aon.at)
to monitor the performance Cisco Unified Communications Manager CUCM.

I have used the Google Go programming language because of no need to install
any libraries.

The plugin uses the Cisco PerfmonPort SOAP Service via HTTPS to do a wide variety of checks.

This nagios plugin is free software, and comes with ABSOLUTELY NO WARRANTY.
It may be used, redistributed and/or modified under the terms of the GNU
General Public Licence (see http://www.fsf.org/licensing/licenses/gpl.txt).

# tested with: 	
		Cisco Unified Communications Manager CUCM version 10.5.1.11901-1
 		Cisco Unified Communications Manager CUCM version 9.1.2.11900-12
 		Cisco Unified Communications Manager CUCM version 8.6.2.22900-9

# see also:
 		Cisco Unified Communications Manager XML Developers Guide, Release 9.0(1)
 		http://www.cisco.com/c/en/us/td/docs/voice_ip_comm/cucm/devguide/9_0_1/xmldev-901.html

# changelog:
		Version 0.1 (15.05.2014) initial release
		Version 0.2 (20.05.2014) object caching added. new func loadStruct and saveStruct
		Version 0.3 (27.02.2015) General Public Licence added
		Version 0.3.1 (27.02.2015) new flag -m maximum cache age in seconds and flag -a and flag -A Cisco AXL API version of AXL XML Namespace
		Version 0.3.2 (27.02.2015) changed flag -H usage description

# Usage
  -A="apiVersion": Cisco AXL API version of AXL XML Namespace
  
  -H="": AXL server IP address
  
  -N="": Node IP address
  
  -V=false: print plugin version
  
  -c="1": Critical threshold or threshold range
  
  -d=0: print debug, level: 1 errors only, 2 warnings and 3 informational messages
  
  -l=false: print PerfmonListCounter
  
  -m=180: maximum cache age in seconds
  
  -n="": Counter name
  
  -o="Memory": Perfmon object with optional tailing instance names in parenthesis
  
  -p="": password
  
  -u="": username
  
  -w="1": Warning threshold or threshold range
