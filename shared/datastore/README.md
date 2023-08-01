# Datastore

## Statistics

```
L1 down 

local cache not found -> 
  no l1 => 
  l1 cache not found -> + noKey
    no l2 =>
    l2 cache not found => + L2NoKey
    l2 cache timeout => + timeout 
    l2 cache error => + timeout + error
    l2 cache found => + L1Copy + hasValue
  l1 cache timeout => + timeout + error
  l1 cache error => + error
  l1 cache found => + hasValue

local cache found ->
  expired -> expired
	no l1 => 
	l1 cache not found -> + noKey
	  no l2 =>
	  l2 cache not found => + L2NoKey
	  l2 cache timeout => + timeout
	  l2 cache error => + timeout + error
	  l2 cache found => + L1Copy + hasValue
	l1 cache timeout => + timeout + error
	l1 cache error => + error
	l1 cache found => + hasValue

  no such value => cacheHit + noKey

  found => cacheHit + hasValue

  collision -> collision
	no l1 => 
	l1 cache not found -> + noKey
	  no l2 =>
	  l2 cache not found => + L2NoKey
	  l2 cache timeout => + timeout
	  l2 cache error => + timeout + error
	  l2 cache found => + L1Copy + hasValue
	l1 cache timeout => + timeout + error
	l1 cache error => + error
	l1 cache found => + hasValue
  
```
