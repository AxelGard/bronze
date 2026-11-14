-- Bronze support for Neovim: tree-sitter highlighting and the language
-- server. This file is editors/nvim/plugin/bronze.lua in the Bronze repo.

if vim.g.loaded_bronze then
  return
end
vim.g.loaded_bronze = true

-- The repository root, three directories up from this file.
local repo = vim.fn.fnamemodify(debug.getinfo(1, "S").source:sub(2), ":p:h:h:h:h")

-- Tree-sitter: tell nvim-treesitter where the grammar is, so that
-- `:TSInstall bronze` builds it. The grammar's src/parser.c is committed,
-- so building needs only a C compiler. (Highlight queries come from
-- queries/bronze/ in this plugin.)
local ok, parsers = pcall(require, "nvim-treesitter.parsers")
if ok and parsers.get_parser_configs then
  parsers.get_parser_configs().bronze = {
    install_info = {
      url = repo,
      location = "editors/tree-sitter-bronze",
      files = { "src/parser.c" },
      generate_requires_npm = false,
      requires_generate_from_grammar = false,
    },
    filetype = "bronze",
  }
end

vim.api.nvim_create_autocmd("FileType", {
  pattern = "bronze",
  group = vim.api.nvim_create_augroup("bronze", { clear = true }),
  callback = function(args)
    pcall(vim.treesitter.start, args.buf, "bronze")
  end,
})

-- The language server: `bronze lsp` (Neovim 0.11 or later).
if vim.lsp.config and vim.fn.executable("bronze") == 1 then
  vim.lsp.config("bronze", {
    cmd = { "bronze", "lsp" },
    filetypes = { "bronze" },
    root_markers = { ".git" },
  })
  vim.lsp.enable("bronze")
end
