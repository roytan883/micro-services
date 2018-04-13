#!/usr/bin/python
# coding=utf-8

import os
import sys
import commands
import re

# commands.getstatusoutput


def callSys(cmdString):
    print "\n"
    print "Start: ", cmdString
    ret = os.system(cmdString)
    if ret != 0:
        print "Error :", cmdString
        exit()
    print "Finish: ", cmdString


def runSys(cmdString):
    print "\n"
    print "Start: ", cmdString
    ret = os.system(cmdString)
    print "Finish: ", cmdString


def writeFile(filePath, content):
    print "\n"
    print "Start writeFile: ", filePath
    fo = open(filePath, "w")
    fo.write(content)
    fo.close()
    print "Finish writeFile: ", filePath


def getInputString(description, canUseDefault, defaultValue):
    print ""
    print "--------------------------------------------------------"
    outValue = ""
    if canUseDefault == True:
        print "请配置 %s:" % description
        useDefault = raw_input("是否使用默认值(%s)? (Y/n) " % defaultValue)
        useDefault = useDefault.strip(' ')
        if useDefault == '' or useDefault == 'Y' or useDefault == 'y':
            outValue = defaultValue
        elif useDefault == 'N' or useDefault == 'n':
            newValue = raw_input("请配置新值: ")
            newValue = newValue.strip(' ')
            if newValue == '':
                print "输入不能为空"
                exit()
            else:
                outValue = newValue
        else:
            print "输入错误111"
            exit()
    else:
        newValue = raw_input("请配置 %s: " % description)
        newValue = newValue.strip(' ')
        if newValue == '':
            print "输入不能为空"
            exit()
        else:
            outValue = newValue
    print ""
    print "%s 已设置为: %s" % (description, outValue)
    print "--------------------------------------------------------"
    return outValue


def getInputNumber(description, canUseDefault, defaultValue, minValue, maxValue):
    print ""
    print "--------------------------------------------------------"
    outValue = 0
    if canUseDefault == True:
        print "请配置 %s:" % description
        useDefault = raw_input("是否使用默认值(%d)? (Y/n) " % defaultValue)
        useDefault = useDefault.strip(' ')
        if useDefault == '' or useDefault == 'Y' or useDefault == 'y':
            outValue = defaultValue
        elif useDefault == 'N' or useDefault == 'n':
            newValue = raw_input("请配置新值(%d-%d): " % (minValue, maxValue))
            newValue = newValue.strip(' ')
            if newValue == '':
                print "输入不能为空"
                exit()
            else:
                outValueInt = int(newValue)
                if outValueInt < minValue or outValueInt > maxValue:
                    print "输入值错误"
                    exit()
                else:
                    outValue = outValueInt
        else:
            print "输入错误"
            exit()
    else:
        newValue = raw_input("请配置 %s(%d-%d): " %
                             (description, minValue, maxValue))
        newValue = newValue.strip(' ')
        if newValue == '':
            print "输入不能为空"
            exit()
        else:
            outValueInt = int(newValue)
            if outValueInt < minValue or outValueInt > maxValue:
                print "输入值错误"
                exit()
            else:
                outValue = outValueInt
    print ""
    print "%s 已设置为: %d" % (description, outValue)
    print "--------------------------------------------------------"
    return outValue


def needYes(description):
    print ""
    print "--------------------------------------------------------"
    outValue = False
    newValue = raw_input("请确认 %s (Y) " % description)
    if newValue == '' or newValue == 'Y' or newValue == 'y':
        outValue = True
    else:
        print "输入错误"
        exit()
    print ""
    print "%s 已设置为: %s" % (description, outValue)
    print "--------------------------------------------------------"
    return outValue

# 获取脚本文件的当前路径


def cur_file_dir():
    # 获取脚本路径
    path = sys.path[0]
    # 判断为脚本文件还是py2exe编译后的文件，如果是脚本文件，则返回的是脚本的目录，如果是py2exe编译后的文件，则返回的是编译后的文件路径
    if os.path.isdir(path):
        return path
    elif os.path.isfile(path):
        return os.path.dirname(path)

# 获取局域网IP


def get_lan_ip():
    ip = os.popen(
        "ifconfig|grep 'inet '|grep -v '0.1'|xargs|awk -F '[ :]' '{print $3}'").readline().rstrip()
    return ip


#========================== 脚本开始 ==========================
thisFileDir = cur_file_dir()
print "脚本目录为: %s" % (thisFileDir)

os.chdir("%s" % (thisFileDir))
workDir = os.getcwd()

os.chdir("./ws-connector")
callSys("go build && ./ws-connector -s nats://127.0.0.1:12008 -p 12220 -i 0 -d 1 -fe 1 -wf 1 > /dev/null 2>&1 &")
os.chdir("../")

os.chdir("./ws-online")
callSys("go build && ./ws-online -s nats://127.0.0.1:12008 -i 0 -d 1 -wf 1 > /dev/null 2>&1 &")
os.chdir("../")

os.chdir("./ws-cache")
callSys("go build && ./ws-cache -s nats://127.0.0.1:12008 -i 0 -d 1 -fe 1 -wf 1 > /dev/null 2>&1 &")
os.chdir("../")

os.chdir("./ws-sender")
callSys("go build && ./ws-sender -s nats://127.0.0.1:12008 -i 0 -d 1 -fe 1 -wf 1 > /dev/null 2>&1 &")
os.chdir("../")
