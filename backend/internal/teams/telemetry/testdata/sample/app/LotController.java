package com.harvest.lots;

@RestController
@RequestMapping("/api/lots")
public class LotController {
    @GetMapping("/{id}")
    Lot get(@PathVariable long id) { return null; }

    @GetMapping("/archive")
    List<Lot> archive() { return null; }
}
