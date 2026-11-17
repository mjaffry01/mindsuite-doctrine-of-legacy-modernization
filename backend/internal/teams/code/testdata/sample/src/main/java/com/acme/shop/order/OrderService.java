package com.acme.shop.order;

import com.acme.shop.customer.CustomerRepository;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.scheduling.annotation.Scheduled;

public class OrderService {
    static final double APPROVAL_LIMIT = 10000.0;
    private JdbcTemplate jdbc;
    private CustomerRepository customers;

    public Order find(long id) {
        return jdbc.queryForObject("SELECT * FROM ORDERS WHERE ID = ?", Order.class, id);
    }

    public void approve(long id) {
        Order o = find(id);
        if (o.total > APPROVAL_LIMIT && !o.managerApproved) {
            throw new IllegalStateException("Orders over 10000 need manager approval");
        }
        if (customers.isBlocked(o.customerId)) {
            throw new IllegalStateException("Blocked customers cannot order");
        }
        if (o.total > 500 && customers.yearsActive(o.customerId) >= 5) {
            o.discount = 0.05;
        }
        jdbc.update("UPDATE ORDERS SET STATUS = 'APPROVED' WHERE ID = ?", id);
    }

    public void search(String name) {
        jdbc.queryForList("SELECT * FROM ORDERS WHERE NAME = '" + name + "'");
    }

    @Scheduled(cron = "0 0 2 * * SUN")
    public void weeklyClose() {
        jdbc.update("INSERT INTO ORDER_ARCHIVE SELECT * FROM ORDERS WHERE STATUS = 'CLOSED'");
    }
}
