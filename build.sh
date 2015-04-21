#!/bin/bash
COMPILERS[1]=go-linux-386
COMPILERS[2]=go
#COMPILERS[3]=go-windows-386
#COMPILERS[4]=go-windows-amd64

for i in {1..2};
do
  COMPILER="${COMPILERS[i]}"

  printf "** Running tests against ${COMPILER}\n"
  "$COMPILER" test github.com/zerklabs/busybody
done

for i in {1..2};
do
  COMPILER="${COMPILERS[i]}"
  printf "** Building against ${COMPILER}\n"
  "$COMPILER" build -a github.com/zerklabs/busybody
done

for i in {1..2};
do
  COMPILER="${COMPILERS[i]}"
  printf "** Installing\n"
  "$COMPILER" install -a github.com/zerklabs/busybody
done
