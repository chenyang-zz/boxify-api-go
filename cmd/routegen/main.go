package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
)

func main() {
	root := flag.String("root", ".", "project root")
	noColor := flag.Bool("no-color", false, "disable colored output")
	flag.Parse()

	report, err := Generate(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	printReport(os.Stdout, report, !*noColor)
}

func Generate(root string) (Report, error) {
	report := Report{Root: root}
	routes, err := scanRoutes(root)
	if err != nil {
		return report, err
	}
	if len(routes) == 0 {
		return report, nil
	}

	handlers, err := scanHandlers(root)
	if err != nil {
		return report, err
	}
	logics, err := scanLogics(root)
	if err != nil {
		return report, err
	}
	requestDTOs, err := scanRequestDTOs(root)
	if err != nil {
		return report, err
	}

	routesByDomain := map[string][]Route{}
	for _, route := range routes {
		routesByDomain[route.Domain] = append(routesByDomain[route.Domain], route)
	}
	domains := make([]string, 0, len(routesByDomain))
	for domain := range routesByDomain {
		domains = append(domains, domain)
	}
	sort.Strings(domains)

	for _, domain := range domains {
		if err := generateHandler(root, domain, routesByDomain[domain], handlers, requestDTOs, &report); err != nil {
			return report, err
		}
		for _, route := range routesByDomain[domain] {
			key := logicKey(route.Domain, route.HandlerMethod)
			if path, ok := logics[key]; ok {
				report.Add(FileSkipped, path)
				continue
			}
			if logicFileExists(root, route) {
				report.Add(FileSkipped, logicPath(root, route))
				continue
			}
			if err := generateLogic(root, route, &report); err != nil {
				return report, err
			}
			logics[key] = logicPath(root, route)
		}
	}
	return report, nil
}
