package com.harvest.export;

@RestController
public class ExportController {
    @GetMapping("/api/export/csv")
    String csv() { return ""; }

    @PostMapping("/api/deliveries")
    void deliver() { }
}
