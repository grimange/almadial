#!/usr/bin/env python3
"""Derive the pinned r4037 FastAGI lab service; no execution or deployment.

Process identity, listener binding and private configuration are isolated.
The reproduced MariaDB strict-mode defect in call_log initial empty numeric/date
values is corrected to NULL/zero, without changing global SQL mode.
"""
import hashlib

SOURCE_SHA256 = 'f376c7fe831ae7b3316ef3af1c1b3a006042a4c92dc876bf5bf990708251ee0a'


def derive(data):
    if hashlib.sha256(data).hexdigest() != SOURCE_SHA256:
        raise ValueError('unreviewed r4037 FastAGI source')
    text = data.decode()
    substitutions = (
        ("'$now_date_epoch','','','','','$callerid')",
         "'$now_date_epoch',NULL,0,0,0,'$callerid')", 1),
        ("'/etc/astguiclient.conf'", "'/etc/astguiclient-m006.conf'", 2),
        ('port=>4577,', "host=>'127.0.0.1',\n\t\t\t\tport=>4577,", 1),
        ("user=>'root',", "user=>'asterisk',", 1),
        ("group=>'root',", "group=>'asterisk',", 1),
    )
    for old, new, count in substitutions:
        if text.count(old) != count:
            raise ValueError('unexpected FastAGI derivation preimage')
        text = text.replace(old, new)
    return text.encode()
