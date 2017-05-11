#!/usr/bin/env python3
#
# Purges a message from Clyde's memory; the full text of the message
# should be provided on stdin followed by EOF.

import json
import sys

prefix_len = 2

words = sys.stdin.read().split()

with open("chain.json") as f:
    chain = json.load(f)

ngram = [""]*(prefix_len-1) + ["START"]

for word in words:
    for i in range(prefix_len + 1):
        if i < prefix_len and ngram[i] == "":
            continue
        key = (" ".join(ngram[i:])).lower()
        if key in chain:
            val = chain[key].get(word, 0)
            if word in chain[key]:
                if chain[key][word] <= 1:
                    print("Deleting {0}: {1}".format(key, word))
                    del chain[key][word]
                    if len(chain[key]) == 0:
                        print ("Deleting key {0}".format(key))
                        del chain[key]
                else:
                    print("Decrementing {0}: {1}".format(key, word))
                    chain[key][word] -= 1
    ngram = ngram[1:] + [word]

with open("chain.json", "w") as f:
    json.dump(chain, f)
