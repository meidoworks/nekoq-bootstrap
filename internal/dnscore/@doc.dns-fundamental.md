# DNS Fundamental

## 1. Lookup methods

* Recursive lookup
* Iterative lookup

## 2. Role of DNS server

* Root
* TLD
* Authoritative
* Recursion/Cache/Forwarding

## 3. Behavior of unknown records

1. Exactly match the name and the type => resolve result
2. Match the name but no such type => empty result
3. Match the name but only NS records found => delegation
    * response content: NS records in authority section + A/AAAA for the NS server in additional section
4. No match => NXDOMAIN

## 4. Fundamental DNS records

1. SOA for authoritative
2. NS

## 5. Subdomain delegation and NS records

* Configure NS records only for subdomains
* Configure SOA for subdomains on child NS server
* Configure subdomains with concrete types on child NS server

