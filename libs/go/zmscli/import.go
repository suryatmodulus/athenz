// Copyright 2016 Yahoo Inc.
// Licensed under the terms of the Apache version 2.0 license. See LICENSE file for terms.

package zmscli

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/AthenZ/athenz/clients/go/zms"
)

func parseRoleMember(memberStruct map[interface{}]interface{}) *zms.RoleMember {
	roleMember := zms.NewRoleMember()
	roleMember.MemberName = zms.MemberName(memberStruct["name"].(string))

	if val, ok := memberStruct["expiration"]; ok {
		expiration := val.(string)
		exprTimeStamp, err := getTimestamp(expiration)
		if err != nil {
			fmt.Println("***", err)
		}
		roleMember.Expiration = &exprTimeStamp
	}

	if val, ok := memberStruct["review"]; ok {
		review := val.(string)
		reviewTimeStamp, err := getTimestamp(review)
		if err != nil {
			fmt.Println("***", err)
		}
		roleMember.ReviewReminder = &reviewTimeStamp
	}

	return roleMember
}

func shouldReportError(commandSkipErrors, clientSkipErrors bool, err error) bool {
	// if we have no error then there is nothing to check
	if err == nil {
		return false
	}
	// if the skip errors argument is false then we're going
	// to report the error
	if !commandSkipErrors {
		return true
	}
	// output the error
	fmt.Println("***", err)
	// if the client requested skip error option is disabled then
	// we're only going to skip any errors where the object already exists
	if clientSkipErrors {
		return false
	}
	return !strings.Contains(err.Error(), "already exists")
}

func (cli Zms) importRoles(dn string, lstRoles []*zms.Role, validatedAdmins []string, skipErrors bool) error {
	for _, role := range lstRoles {
		rn := localName(string(role.Name), ":role.")
		_, _ = fmt.Fprintf(os.Stdout, "Processing role "+rn+"...\n")
		if len(role.RoleMembers) > 0 {
			roleMembers := make([]*zms.RoleMember, 0)
			var err error
			if rn == "admin" && validatedAdmins != nil {
				// need to retrieve the current admin role
				// and make sure to remove any existing admin
				adminRole, err := cli.Zms.GetRole(zms.DomainName(dn), "admin", nil, nil, nil)
				if err != nil {
					return err
				}
				for _, roleMember := range role.RoleMembers {
					if !cli.containsMember(adminRole.RoleMembers, string(roleMember.MemberName)) {
						roleMembers = append(roleMembers, roleMember)
					}
				}
				for _, admin := range validatedAdmins {
					roleMember := zms.NewRoleMember()
					roleMember.MemberName = zms.MemberName(admin)
					if !cli.containsMember(roleMembers, admin) && !cli.containsMember(adminRole.RoleMembers, admin) {
						roleMembers = append(roleMembers, roleMember)
					}
				}
				_, err = cli.AddRoleMembers(dn, rn, roleMembers)
			} else {
				for _, roleMember := range role.RoleMembers {
					roleMembers = append(roleMembers, roleMember)
				}
				b := cli.Verbose
				cli.Verbose = true
				_, err = cli.AddRegularRole(dn, rn, roleMembers)
				cli.Verbose = b
			}
			if shouldReportError(skipErrors, cli.SkipErrors, err) {
				return err
			}
		} else if role.Trust != "" {
			trust := string(role.Trust)
			_, err := cli.AddDelegatedRole(dn, rn, trust)
			if shouldReportError(skipErrors, cli.SkipErrors, err) {
				return err
			}
		} else {
			roleMembers := make([]*zms.RoleMember, 0)
			b := cli.Verbose
			cli.Verbose = true
			_, err := cli.AddRegularRole(dn, rn, roleMembers)
			cli.Verbose = b
			if shouldReportError(skipErrors, cli.SkipErrors, err) {
				return err
			}
		}
	}
	return nil
}

func (cli Zms) importRolesOld(dn string, lstRoles []interface{}, validatedAdmins []string, skipErrors bool) error {
	for _, role := range lstRoles {
		roleMap := role.(map[interface{}]interface{})
		rn := roleMap["name"].(string)
		_, _ = fmt.Fprintf(os.Stdout, "Processing role "+rn+"...\n")
		if val, ok := roleMap["members"]; ok {
			mem := val.([]interface{})
			roleMembers := make([]*zms.RoleMember, 0)
			var err error
			if rn == "admin" && validatedAdmins != nil {
				// need to retrieve the current admin role
				// and make sure to remove any existing admin
				role, err := cli.Zms.GetRole(zms.DomainName(dn), "admin", nil, nil, nil)
				if err != nil {
					return err
				}
				for _, mbr := range mem {
					roleMember := parseRoleMember(mbr.(map[interface{}]interface{}))
					if !cli.containsMember(role.RoleMembers, string(roleMember.MemberName)) {
						roleMembers = append(roleMembers, roleMember)
					}
				}
				for _, admin := range validatedAdmins {
					roleMember := zms.NewRoleMember()
					roleMember.MemberName = zms.MemberName(admin)
					if !cli.containsMember(roleMembers, admin) && !cli.containsMember(role.RoleMembers, admin) {
						roleMembers = append(roleMembers, roleMember)
					}
				}
				_, err = cli.AddRoleMembers(dn, rn, roleMembers)
			} else {
				for _, m := range mem {
					roleMember := parseRoleMember(m.(map[interface{}]interface{}))
					roleMembers = append(roleMembers, roleMember)
				}
				b := cli.Verbose
				cli.Verbose = true
				_, err = cli.AddRegularRole(dn, rn, roleMembers)
				cli.Verbose = b
			}
			if shouldReportError(skipErrors, cli.SkipErrors, err) {
				return err
			}
		} else if val, ok := roleMap["trust"]; ok {
			trust := val.(string)
			_, err := cli.AddDelegatedRole(dn, rn, trust)
			if shouldReportError(skipErrors, cli.SkipErrors, err) {
				return err
			}
		} else {
			roleMembers := make([]*zms.RoleMember, 0)
			b := cli.Verbose
			cli.Verbose = true
			_, err := cli.AddRegularRole(dn, rn, roleMembers)
			cli.Verbose = b
			if shouldReportError(skipErrors, cli.SkipErrors, err) {
				return err
			}
		}
	}
	return nil
}

func (cli Zms) importPolicies(dn string, lstPolicies []*zms.Policy, skipErrors bool) error {
	for _, policy := range lstPolicies {
		name := localName(string(policy.Name), ":policy.")
		_, _ = fmt.Fprintf(os.Stdout, "Processing policy "+name+"...\n")
		assertions := make([]*zms.Assertion, 0)
		if len(policy.Assertions) == 0 {
			_, _ = fmt.Fprintf(os.Stdout, "Skipping empty policy: "+name+"\n")
			continue
		}
		for _, a := range policy.Assertions {
			if name == "admin" && a.Role == "admin" && a.Action == "*" && a.Resource == "*" {
				continue
			}

			assertions = append(assertions, a)
		}

		if name != "admin" {
			_, err := cli.AddPolicyWithAssertions(dn, name, assertions)
			if shouldReportError(skipErrors, cli.SkipErrors, err) {
				return err
			}
		}
	}
	return nil
}

func (cli Zms) importPoliciesOld(dn string, lstPolicies []interface{}, skipErrors bool) error {
	for _, policy := range lstPolicies {
		policyMap := policy.(map[interface{}]interface{})
		name := policyMap["name"].(string)
		_, _ = fmt.Fprintf(os.Stdout, "Processing policy "+name+"...\n")
		assertions := make([]*zms.Assertion, 0)
		if val, ok := policyMap["assertions"]; ok {
			if val == nil {
				_, _ = fmt.Fprintf(os.Stdout, "Skipping empty policy: "+name+"\n")
				continue
			}
			lst := val.([]interface{})
			if len(lst) > 0 {
				for _, a := range lst {
					if name == "admin" && a.(string) == "grant * to admin on *" {
						continue
					}
					assertion := strings.Split(a.(string), " ")
					newAssertion, err := parseAssertion(dn, assertion)
					if shouldReportError(skipErrors, cli.SkipErrors, err) {
						return err
					}
					assertions = append(assertions, newAssertion)
				}
			}
		}
		if name != "admin" {
			_, err := cli.AddPolicyWithAssertions(dn, name, assertions)
			if shouldReportError(skipErrors, cli.SkipErrors, err) {
				return err
			}
		}
	}
	return nil
}

func (cli Zms) generatePublicKeys(lstPublicKeys []interface{}) []*zms.PublicKeyEntry {
	publicKeys := make([]*zms.PublicKeyEntry, 0)
	for _, pubKey := range lstPublicKeys {
		publicKeyMap := pubKey.(map[interface{}]interface{})
		// if we're using just version numbers then yaml
		// will interpret the key id as integer
		var keyID string
		switch v := publicKeyMap["keyID"].(type) {
		case int:
			keyID = strconv.Itoa(v)
		case string:
			keyID = v
		default:
			panic("Unknown data type for keyid")
		}
		value := publicKeyMap["value"].(string)
		publicKey := zms.PublicKeyEntry{
			Key: value,
			Id:  keyID,
		}
		publicKeys = append(publicKeys, &publicKey)
	}
	return publicKeys
}

func (cli Zms) importServices(dn string, lstServices []*zms.ServiceIdentity, skipErrors bool) error {
	for _, service := range lstServices {
		name := string(service.Name)
		_, _ = fmt.Fprintf(os.Stdout, "Processing service "+name+"...\n")
		publicKeys := service.PublicKeys
		_, err := cli.AddServiceWithKeys(dn, name, publicKeys)
		if shouldReportError(skipErrors, cli.SkipErrors, err) {
			return err
		}
		if service.ProviderEndpoint != "" {
			_, err = cli.SetServiceEndpoint(dn, name, service.ProviderEndpoint)
			if shouldReportError(skipErrors, cli.SkipErrors, err) {
				return err
			}
		}
		if service.User != "" || service.Group != "" || service.Executable != "" {
			_, err = cli.SetServiceExe(dn, name, service.Executable, service.User, service.Group)
			if shouldReportError(skipErrors, cli.SkipErrors, err) {
				return err
			}
		}
		if len(service.Hosts) > 0 {
			_, err = cli.AddServiceHost(dn, name, service.Hosts)
			if shouldReportError(skipErrors, cli.SkipErrors, err) {
				return err
			}
		}
	}
	return nil
}

func (cli Zms) importServicesOld(dn string, lstServices []interface{}, skipErrors bool) error {
	for _, service := range lstServices {
		serviceMap := service.(map[interface{}]interface{})
		name := serviceMap["name"].(string)
		_, _ = fmt.Fprintf(os.Stdout, "Processing service "+name+"...\n")
		var lstPublicKeys []interface{}
		if val, ok := serviceMap["publicKeys"]; ok {
			lstPublicKeys = val.([]interface{})
		}
		publicKeys := cli.generatePublicKeys(lstPublicKeys)
		_, err := cli.AddServiceWithKeys(dn, name, publicKeys)
		if err != nil {
			return err
		}
		if val, ok := serviceMap["providerEndpoint"]; ok {
			endpoint := val.(string)
			if endpoint != "" {
				_, err = cli.SetServiceEndpoint(dn, name, endpoint)
				if shouldReportError(skipErrors, cli.SkipErrors, err) {
					return err
				}
			}
		}
		user := ""
		if val, ok := serviceMap["user"]; ok {
			user = val.(string)
		}
		group := ""
		if val, ok := serviceMap["group"]; ok {
			group = val.(string)
		}
		exe := ""
		if val, ok := serviceMap["executable"]; ok {
			exe = val.(string)
		}
		if user != "" || group != "" || exe != "" {
			_, err = cli.SetServiceExe(dn, name, exe, user, group)
			if shouldReportError(skipErrors, cli.SkipErrors, err) {
				return err
			}
		}
		if val, ok := serviceMap["hosts"]; ok {
			hostList := val.([]interface{})
			hosts := make([]string, 0)
			for _, host := range hostList {
				hosts = append(hosts, host.(string))
			}
			_, err = cli.AddServiceHost(dn, name, hosts)
			if shouldReportError(skipErrors, cli.SkipErrors, err) {
				return err
			}
		}
	}
	return nil
}
