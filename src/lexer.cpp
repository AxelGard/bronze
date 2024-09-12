#include "lexer.hh"

std::vector<std::string> split(std::string str, char on)
{
    std::vector<std::string> words;
    std::string word{};
    for (char ch : str)
    {
        if (ch != ' ')
        {
            word += ch;
        } 
        else if (!word.empty())
        {
            words.push_back(word);
            words.clear();
        }
    }
    if (!word.empty())
    {
        words.push_back(word);
    }
    return words;
}

std::vector<Token> tokenize(std::string& sourceCode)
{
    std::vector<Token> tokens;
    std::vector<std::string> src = split(sourceCode);
    while (! src.empty())
    {
        std::string t = src.front();
        if ( t == "(")
        {
            tokens.push_back(Token(t, TokenType::OpenParen));
        }
        else if (t == ")")
        {
            tokens.push_back(Token(t, TokenType::CloseParan));
        }
        else if (t == ")")
        {
            tokens.push_back(Token(t, TokenType::CloseParan));
        }

        src.erase(src.begin());
    }

    return tokens;
}
