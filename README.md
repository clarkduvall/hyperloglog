[![Build Status](https://travis-ci.org/clarkduvall/hyperloglog.svg?branch=master)](https://travis-ci.org/clarkduvall/hyperloglog) [![Coverage Status](https://coveralls.io/repos/clarkduvall/hyperloglog/badge.png?branch=master)](https://coveralls.io/r/clarkduvall/hyperloglog?branch=master)
# HyperLogLog and HyperLogLog++
Implements the HyperLogLog and HyperLogLog++ algorithms.

HyperLogLog paper: http://algo.inria.fr/flajolet/Publications/FlFuGaMe07.pdf

HyperLogLog++ paper: http://research.google.com/pubs/pub40671.html

## Documentation
Documentation can be found [here](http://godoc.org/github.com/clarkduvall/hyperloglog).

## Comparison of Algorithms
The HyperLogLog++ algorithm has much lower error for small cardinalities. This
is because it uses a different representation of data for small sets of data.
Data generated using this library shows the difference for N < 10000:

![N < 10000](10000.png)

HyperLogLog++ also has bias correction which helps offset estimation errors in
the original HyperLogLog algorithm. This correction can be seen here, again
using data generated using this library:

![N < 80000](80000.png)

## Future Improvements
- Right now HLL++ uses 8 bits per register. It could use 6 bits and take less
  memory.
- The list compression algorithm could be improved, allowing the sparse
  representation to be used longer.
- Add Merge() methods for combining two HLLs.
