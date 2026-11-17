package com.acme.shop.order;

import com.acme.shop.billing.InvoiceService;
import org.springframework.web.bind.annotation.*;
import org.springframework.security.access.prepost.PreAuthorize;

@RestController
@RequestMapping("/api/orders")
public class OrderController {
    private final OrderService service;
    private final InvoiceService invoices;

    public OrderController(OrderService service, InvoiceService invoices) { this.service = service; this.invoices = invoices; }

    @GetMapping("/{id}")
    public Order get(@PathVariable long id) { return service.find(id); }

    @PreAuthorize("hasRole('MANAGER')")
    @PostMapping("/{id}/approve")
    public void approve(@PathVariable long id) { service.approve(id); }
}
