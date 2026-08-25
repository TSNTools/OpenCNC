#!/usr/bin/env python3
import sys


"""
Script to automatically add logging to the given files.

The first argument given is the name of the service. Every argument following that is the file to add logging to.
A backup file is also made called <filename>.backup. This is done just in case the adding of the logger doesn't work as intended.
This might happen if a function declaration not accounted for is found.
"""

def get_package(lines):
    for l in lines:
        if l.startswith("package"):
            return l.split(" ")[1].strip()

def add_logging(lines, filename, service_name):

    name = filename.split('/')[-1].split('.')[0]
    package = get_package(lines)

    out = []
    other = lines.copy()
    while len(other) > 0:
        line = other.pop(0)
        out.append(line)
        if line.startswith("func") == False:
            continue
        if line.find("}") != -1: # if the f
            continue
        funcname = line.split("(")[0].split(" ")[-1]
        count = 1
        #This is the loop to count and find the end of the function
        out.append("\tlog_id_dont_collide := time.Now().UnixMicro(); logger//log.(\"%s.%s.%s.%s.Start_\" + fmt.Sprint(log_id_dont_collide) + \".\" + fmt.Sprint(log_id_dont_collide))\n" % (service_name, package, name, funcname))
        ret = False
        tabs = "\t"
        while count > 0 and len(other) > 0:
            line = other.pop(0)
            if line.find("{") != -1:
                count += 1
            if line.find("}") != -1:
                count -= 1
            # Need to add for every instance of return.
            if line.find("return") != -1:
                out.append("%slogger//log.(\"%s.%s.%s.%s.End_\" +  fmt.Sprint(log_id_dont_collide) + \".\" + fmt.Sprint(time.Now().UnixMicro()))\n" % (tabs, service_name, package, name, funcname))
                ret = True #Used to keeping track if the last value was a return statement
                out.append(line)
                continue

            elif count == 0:
                if not ret:
                    out.append("%slogger//log.(\"%s.%s.%s.%s.End_\" +  fmt.Sprint(log_id_dont_collide) + \".\" + fmt.Sprint(time.Now().UnixMicro()))\n" % (tabs, service_name, package, name, funcname))
            ret = False
            out.append(line)
            tabs = "\t" * (line.count("\t"))

    return out


for i in range(2, len(sys.argv)):
    lines = []
    with open(sys.argv[i], "r") as f:
        lines = f.readlines()

    # Write a backup file just in case something goes wrong
    with open(sys.argv[i] + ".backup", "w") as f:
        f.write("".join(lines))

    
    #Remove old logging
    lines = [l for l in lines if l.find("logger//log.") == -1]

    lines = add_logging(lines, sys.argv[i], sys.argv[1])

    with open(sys.argv[i], "w") as f:
        f.write("".join(lines))

