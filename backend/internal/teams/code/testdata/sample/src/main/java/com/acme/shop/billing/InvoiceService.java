package com.acme.shop.billing;

import com.acme.shop.order.Order;
import org.springframework.jdbc.core.JdbcTemplate;
import java.security.MessageDigest;

public class InvoiceService {
    private JdbcTemplate jdbc;
    public void invoice(Order o) throws Exception {
        jdbc.update("UPDATE ORDERS SET INVOICED = 1 WHERE ID = ?", o.id);
        jdbc.update("INSERT INTO INVOICES (ORDER_ID, AMOUNT) VALUES (?, ?)", o.id, o.total);
        MessageDigest.getInstance("MD5");
        new java.net.URL("https://payments.partnerbank.com/api/pay");
    }
}
