resource "azurerm_kubernetes_cluster" "harvest" {
  name                = "aks-harvest"
  location            = "westeurope"
  resource_group_name = "rg-harvest"
}

resource "azurerm_linux_virtual_machine" "jump" {
  name           = "jump-01"
  admin_username = "harvestadmin"
  admin_password = "Sup3rS3cret!x"
}
