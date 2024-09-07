#include <cstdint>


enum class ReservedToken : std::uint8_t
{
    Unkown,

    OpPerenthesisOpen,
    OpPerenthesisClose,
    OpSubscriptOpen,
    OpSubscriptClose,

    OpAdd,
    OpSub,
    OpDiv,
    OpMulti,
    OpExp,


    // keywords 
    KwAnd,
    KwAs,
    KwBreak,
    KwClass,
    KwContinue,
    KwDef,
    KwDel,
    KwIf,
    KwElif,
    KwElse,


    Count
};