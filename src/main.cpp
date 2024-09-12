#include "main.hh"


int main(int argc, char *argv[]){
    
    if (argc == 1) return 0;
    
    std::string file(*(argv+1));
    std::ifstream input(file);
    for (std::string line; std::getline(input, line);){
        std::cout << line << std::endl;
    }
    return 0;
}
