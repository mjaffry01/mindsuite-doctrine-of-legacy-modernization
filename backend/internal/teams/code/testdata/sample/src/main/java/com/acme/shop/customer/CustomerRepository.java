package com.acme.shop.customer;

import org.springframework.jdbc.core.JdbcTemplate;

public class CustomerRepository {
    private JdbcTemplate jdbc;
    public boolean isBlocked(long id) { return jdbc.queryForObject("SELECT BLOCKED FROM CUSTOMERS WHERE ID = ?", Boolean.class, id); }
    public int yearsActive(long id) { return jdbc.queryForObject("SELECT YEARS FROM CUSTOMERS WHERE ID = ?", Integer.class, id); }
}
