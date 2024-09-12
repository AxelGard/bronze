#ifndef UTIL_H
#define UTIL_H 
#include <string>
#include <vector>
#include <sstream>
#include <fstream>

std::vector<std::string> split(std::string str, char on = ' ');
bool isNumber(const std::string& s);

#endif