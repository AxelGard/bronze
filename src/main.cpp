#include "main.hh"


int main(int argc, char *argv[]){
    
    if (argc == 1) return 0;
    
    std::string file(*(argv+1));
    std::ifstream input(file);
    for (std::string line; std::getline(input, line);){
        for (Token t : tokenize(line))
        {
            std::cout << "(" << t.type << ")"; 
        }
        std::cout << std::endl;
    }
    return 0;
}
