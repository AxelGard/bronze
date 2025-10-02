#include "ast.h"
#include "lexer.h"
#include <llvm-14/llvm/Support/ErrorHandling.h>

namespace bronze
{

namespace
{

std::string_view getOpStr(bronze::TokenKind op)
{
    switch (op)
    {
    case bronze::TokenKind::Plus:
        return "+";
    case bronze::TokenKind::Minus:
        return "-";
    case bronze::TokenKind::Asterisk:
        return "*";
    case bronze::TokenKind::Slash:
        return "/";
    case bronze::TokenKind::Equivalent:
        return "==";
    case bronze::TokenKind::And:
        return "&&";
    case bronze::TokenKind::Or:
        return "||";
    case bronze::TokenKind::Lt:
        return "<";
    case bronze::TokenKind::Gt:
        return ">";
    case bronze::TokenKind::Excl:
        return "!";
    }
    llvm_unreachable("unexpected operator");
}

std::string indent(size_t level)
{
    return std::string(level * 4, ' ');
}
} // namespace

} // namespace bronze
