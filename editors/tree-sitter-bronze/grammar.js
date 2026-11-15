/**
 * Tree-sitter grammar for Bronze 0.2 (doc/spec-0.2.md), for editor
 * highlighting.
 *
 * It is looser than the real parser: newlines are ignored, so statements
 * are simply one after the other. That is enough to highlight, and keeps
 * the grammar small. The one place a newline matters is a bare `return` at
 * the end of a line, which must not take the next line as its value: that
 * is a separate token.
 */

const PREC = {
  else: 1,
  or: 2,
  and: 3,
  compare: 4,
  range: 5,
  add: 6,
  multiply: 7,
  unary: 8,
  postfix: 9,
};

const commaSep1 = (rule) => seq(rule, repeat(seq(',', rule)));
const commaSep = (rule) => optional(commaSep1(rule));
// like commaSep, but a `,` may end the list
const trailingCommaSep = (rule) => optional(seq(commaSep1(rule), optional(',')));

module.exports = grammar({
  name: 'bronze',

  extras: ($) => [/\s/, $.comment],

  word: ($) => $.identifier,

  conflicts: ($) => [
    // `else (e) return` binds e; `else (x)` is a fallback value
    [$._expression, $.else_handler],
    // `=> {}` in a match arm is a block (see match_arm)
    [$.block, $.map],
  ],

  rules: {
    source_file: ($) => repeat($._declaration),

    _declaration: ($) =>
      choice($.function_declaration, $.class_declaration, $.enum_declaration),

    // ------------------------------------------------------------------
    // Declarations

    // `static func` is a method of the class, and `mut func` marks a method
    // that changes `this`.
    function_declaration: ($) =>
      seq(
        optional($.visibility),
        optional('static'),
        optional('override'),
        optional('mut'),
        'func',
        field('name', $.identifier),
        field('parameters', $.parameter_list),
        optional(seq('->', field('result', $.return_type))),
        field('body', $.block),
      ),

    parameter_list: ($) => seq('(', commaSep($.parameter), ')'),

    // `name: mut T` may change the caller's value; a plain one is read only.
    parameter: ($) =>
      seq(field('name', $.identifier), ':', optional('mut'), field('type', $.type)),

    return_type: ($) =>
      choice(seq($.type, optional(seq('!', $.type))), seq('!', $.type)),

    type: ($) =>
      seq(
        field('name', $.identifier),
        optional(seq('<', commaSep1($.type), '>')),
        optional('?'),
      ),

    class_declaration: ($) =>
      seq(
        'class',
        field('name', $.identifier),
        optional(seq('use', commaSep1(field('use', $.identifier)))),
        field('body', $.class_body),
      ),

    class_body: ($) =>
      seq('{', repeat(choice($.field_declaration, $.function_declaration)), '}'),

    visibility: (_) => choice('private', 'protected'),

    field_declaration: ($) =>
      seq(
        optional($.visibility),
        choice('con', 'mut'),
        field('name', $.identifier),
        optional(seq(':', field('type', $.type))),
        '=',
        field('value', $._expression),
      ),

    enum_declaration: ($) =>
      seq(
        'enum',
        field('name', $.identifier),
        optional(seq(':', field('type', $.type))),
        field('body', $.enum_body),
      ),

    enum_body: ($) =>
      seq('{', repeat(choice($.enum_case, $.function_declaration)), '}'),

    // `Admin = "admin"` in an enum with values.
    enum_case: ($) =>
      seq(field('name', $.identifier), optional(seq('=', field('value', $._expression)))),

    // ------------------------------------------------------------------
    // Statements

    block: ($) => seq('{', repeat($._statement), '}'),

    _statement: ($) =>
      choice(
        $.variable_declaration,
        $.assignment,
        $.compound_assignment,
        $.if_statement,
        $.while_statement,
        $.for_statement,
        $.break_statement,
        $.continue_statement,
        $.return_statement,
        $.expression_statement,
      ),

    variable_declaration: ($) =>
      seq(
        choice('con', 'mut'),
        field('name', $.identifier),
        optional(seq(':', field('type', $.type))),
        '=',
        field('value', $._expression),
      ),

    assignment: ($) =>
      seq(
        field('left', choice($.identifier, $.member_expression, $.index_expression)),
        '=',
        field('right', $._expression),
      ),

    compound_assignment: ($) =>
      seq(
        field('left', choice($.identifier, $.member_expression, $.index_expression)),
        field('operator', choice('+=', '-=', '*=', '/=', '%=')),
        field('right', $._expression),
      ),

    if_statement: ($) =>
      prec.right(
        seq(
          'if',
          field('condition', $._expression),
          field('consequence', $.block),
          optional(seq('else', field('alternative', choice($.block, $.if_statement)))),
        ),
      ),

    while_statement: ($) =>
      seq('while', field('condition', $._expression), field('body', $.block)),

    for_statement: ($) =>
      seq(
        'for',
        field('variable', $.identifier),
        optional(seq(',', field('variable', $.identifier))),
        'in',
        field('value', $._expression),
        field('body', $.block),
      ),

    break_statement: (_) => 'break',

    continue_statement: (_) => 'continue',

    return_statement: ($) => $._return,

    // `return` at the end of a line has no value.
    _return: ($) =>
      prec.right(choice(alias($._bare_return, 'return'), seq('return', optional($._expression)))),

    _bare_return: (_) => token(prec(1, /return[ \t]*(\/\/[^\n]*)?\r?\n/)),

    expression_statement: ($) => $._expression,

    // ------------------------------------------------------------------
    // Expressions

    _expression: ($) =>
      choice(
        $.else_expression,
        $.binary_expression,
        $.range_expression,
        $.unary_expression,
        $.is_expression,
        $.call_expression,
        $.member_expression,
        $.index_expression,
        $.slice_expression,
        $.match_expression,
        $.list,
        $.map,
        $.parenthesized_expression,
        $.identifier,
        $.integer,
        $.float,
        $.string,
        $.true,
        $.false,
        $.null,
        $.this,
        $.super,
      ),

    else_expression: ($) =>
      prec.right(PREC.else, seq(field('value', $._expression), 'else', $.else_handler)),

    else_handler: ($) =>
      prec.right(
        choice(
          $._return,
          prec.dynamic(1, seq('(', field('error', $.identifier), ')', $._return)),
          prec.dynamic(1, seq('(', field('error', $.identifier), ')', $.block)),
          field('fallback', $._expression),
        ),
      ),

    binary_expression: ($) =>
      choice(
        ...[
          ['or', PREC.or],
          ['and', PREC.and],
          ['==', PREC.compare],
          ['!=', PREC.compare],
          ['<', PREC.compare],
          ['<=', PREC.compare],
          ['>', PREC.compare],
          ['>=', PREC.compare],
          ['+', PREC.add],
          ['-', PREC.add],
          ['*', PREC.multiply],
          ['/', PREC.multiply],
          ['%', PREC.multiply],
        ].map(([op, p]) =>
          prec.left(p, seq(field('left', $._expression), field('operator', op), field('right', $._expression))),
        ),
      ),

    range_expression: ($) =>
      prec.left(
        PREC.range,
        seq(field('start', $._expression), field('operator', choice('..', '..=')), field('end', $._expression)),
      ),

    unary_expression: ($) =>
      prec(PREC.unary, seq(field('operator', choice('not', '-')), field('operand', $._expression))),

    is_expression: ($) =>
      prec.left(PREC.compare, seq(field('value', $._expression), 'is', field('class', $.identifier))),

    call_expression: ($) =>
      prec(PREC.postfix, seq(field('function', $._expression), field('arguments', $.argument_list))),

    argument_list: ($) => seq('(', commaSep($._expression), ')'),

    member_expression: ($) =>
      prec(PREC.postfix, seq(field('object', $._expression), '.', field('member', $.identifier))),

    index_expression: ($) =>
      prec(PREC.postfix, seq(field('object', $._expression), '[', field('index', $._expression), ']')),

    // `xs[a..b]` parses as an index with a range; these are the slices that
    // leave out a bound.
    slice_expression: ($) =>
      prec(
        PREC.postfix,
        seq(
          field('object', $._expression),
          '[',
          choice(
            seq(field('start', $._expression), field('operator', '..')),
            seq(field('operator', choice('..', '..=')), optional(field('end', $._expression))),
          ),
          ']',
        ),
      ),

    list: ($) => seq('[', trailingCommaSep($._expression), ']'),

    map: ($) => seq('{', trailingCommaSep($.map_entry), '}'),

    map_entry: ($) => seq(field('key', $._expression), ':', field('value', $._expression)),

    match_expression: ($) =>
      seq('match', field('value', $._expression), '{', repeat($.match_arm), '}'),

    // `=> {` is always a block, so a map value is written `=> ({})`.
    match_arm: ($) =>
      seq(field('pattern', $.pattern), '=>', field('value', choice($._expression, prec.dynamic(1, $.block)))),

    pattern: ($) =>
      choice(
        $.integer,
        seq('-', $.integer),
        $.string,
        $.true,
        $.false,
        seq(field('enum', $.identifier), '.', field('case', $.identifier)),
        $.identifier, // `_`
      ),

    parenthesized_expression: ($) => seq('(', $._expression, ')'),

    // ------------------------------------------------------------------
    // Tokens

    identifier: (_) => /[\p{L}_][\p{L}\p{Nd}_]*/u,

    integer: (_) => /\d+(_\d+)*/,

    float: (_) => /\d+(_\d+)*(\.\d+(_\d+)*([eE][+-]?\d+)?|[eE][+-]?\d+)/,

    string: ($) =>
      seq('"', repeat(choice($.escape_sequence, $.interpolation, token.immediate(/[^"\\\n{]+/))), '"'),

    interpolation: ($) => seq('{', $._expression, '}'),

    escape_sequence: (_) => token.immediate(/\\[nt"\\{]/),

    true: (_) => 'true',
    false: (_) => 'false',
    null: (_) => 'null',
    this: (_) => 'this',

    // `super(Animal)`, before `.method(...)`.
    super: ($) => seq('super', '(', field('class', $.identifier), ')'),

    comment: (_) =>
      token(choice(seq('//', /[^\n]*/), seq('/*', /[^*]*\*+([^/*][^*]*\*+)*/, '/'))),
  },
});
