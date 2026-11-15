-- Bronze for Neovim: highlighting with the tree-sitter parser, and the
-- language server. `bronze setup nvim` installs this file, together with
-- the parser and its highlight queries, as a package in Neovim's data
-- directory. Needs Neovim 0.11 or later; nvim-treesitter is not needed.

if vim.g.loaded_bronze then
  return
end
vim.g.loaded_bronze = true

-- The package directory: plugin/ is in it.
local package_dir = vim.fn.fnamemodify(debug.getinfo(1, "S").source:sub(2), ":p:h:h")

local function warn_once(key, msg)
  if vim.g["bronze_warned_" .. key] then
    return
  end
  vim.g["bronze_warned_" .. key] = true
  vim.schedule(function()
    vim.notify("bronze: " .. msg, vim.log.levels.WARN)
  end)
end

-- Tells when the parser was set up by another version of bronze than the
-- one on the PATH, so that the grammar may be out of date.
local function check_version()
  local f = io.open(package_dir .. "/version.txt")
  if not f or vim.fn.executable("bronze") ~= 1 then
    return
  end
  local installed = vim.trim(f:read("*a"))
  f:close()
  vim.system({ "bronze", "version" }, { text = true }, function(out)
    local current = vim.trim(out.stdout or "")
    if out.code == 0 and current ~= installed then
      warn_once("version", "the Neovim support was set up by " .. installed .. ", but " .. current
        .. " is installed; run `bronze setup nvim` to update it")
    end
  end)
end

vim.api.nvim_create_autocmd("FileType", {
  pattern = "bronze",
  group = vim.api.nvim_create_augroup("bronze", { clear = true }),
  callback = function(args)
    vim.bo[args.buf].commentstring = "// %s"
    if not pcall(vim.treesitter.start, args.buf, "bronze") then
      warn_once("parser", "the Bronze parser is not installed; run `bronze setup nvim`")
    end
    if vim.fn.executable("bronze") ~= 1 then
      warn_once("path", "`bronze` is not on the PATH, so the language server can't start")
    end
    if not vim.g.bronze_version_checked then
      vim.g.bronze_version_checked = true
      check_version()
    end
  end,
})

-- The language server: `bronze lsp`.
if vim.lsp.config and vim.fn.executable("bronze") == 1 then
  vim.lsp.config("bronze", {
    cmd = { "bronze", "lsp" },
    filetypes = { "bronze" },
    root_markers = { ".git" },
  })
  vim.lsp.enable("bronze")

  -- Completion as you type, with Neovim's own LSP completion. Skipped when
  -- a completion plugin (nvim-cmp, blink.cmp) is loaded, since it already
  -- asks the server; set vim.g.bronze_autocomplete = false to turn it off.
  vim.api.nvim_create_autocmd("LspAttach", {
    group = vim.api.nvim_create_augroup("bronze_lsp", { clear = true }),
    callback = function(args)
      local client = vim.lsp.get_client_by_id(args.data.client_id)
      if not client or client.name ~= "bronze" or vim.g.bronze_autocomplete == false then
        return
      end
      if package.loaded["cmp"] or package.loaded["blink.cmp"] or not vim.lsp.completion then
        return
      end
      vim.lsp.completion.enable(true, client.id, args.buf, { autotrigger = true })
    end,
  })
end
