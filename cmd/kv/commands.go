package kv

import (
	"fmt"
	"github.com/spf13/cobra"
	"strconv"
)

var (
	setCmd = &cobra.Command{
		Use:   "set [key] [value]",
		Short: "Sets the value for a key",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			value := args[1]
			if err := rpcStore.Set(key, []byte(value)); err != nil {
				return err
			} else {
				fmt.Println("set successfully")
			}
			return nil
		},
	}
	setECmd = &cobra.Command{
		Use:   "setE [key] [value] [expireIn] [deleteIn]",
		Short: "Sets the value for a key with expiration and deletion time",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			value := args[1]
			expireIn, err := strconv.ParseUint(args[2], 10, 64)
			if err != nil {
				return fmt.Errorf("expireIn must be a number: %w", err)
			}
			deleteIn, err := strconv.ParseUint(args[3], 10, 64)
			if err != nil {
				return fmt.Errorf("deleteIn must be a number: %w", err)
			}
			if err := rpcStore.SetE(
				key,
				[]byte(value),
				expireIn,
				deleteIn,
			); err != nil {
				return err
			} else {
				fmt.Println("setE successfully")
			}
			return nil
		},
	}
	setEIfUnsetCmd = &cobra.Command{
		Use:   "setEIfUnset [key] [value] [expireIn] [deleteIn]",
		Short: "Sets the value for a key with expiration and deletion time if the key is not already set",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			value := args[1]
			expireIn, err := strconv.ParseUint(args[2], 10, 64)
			if err != nil {
				return fmt.Errorf("expireIn must be a number: %w\n", err)
			}
			deleteIn, err := strconv.ParseUint(args[3], 10, 64)
			if err != nil {
				return fmt.Errorf("deleteIn must be a number: %w", err)
			}
			if err := rpcStore.SetEIfUnset(
				key,
				[]byte(value),
				expireIn,
				deleteIn,
			); err != nil {
				return err
			} else {
				fmt.Println("setEIfUnset successfully")
			}
			return nil
		},
	}
	getCmd = &cobra.Command{
		Use:   "get [key]",
		Short: "Reads the value for a key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			if resp, ok, err := rpcStore.Get(key); err != nil {
				return err
			} else {
				fmt.Printf("key=%s, found=%v, resp=%s\n", key, ok, resp)
			}
			return nil
		},
	}
	exprCmd = &cobra.Command{
		Use:   "expr [key]",
		Short: "Expires the value for a key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			if err := rpcStore.Expire(key); err != nil {
				return err
			} else {
				fmt.Println("expire successfully")
			}
			return nil
		},
	}
	delCmd = &cobra.Command{
		Use:   "del [key]",
		Short: "Deletes a key value pair",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			if err := rpcStore.Delete(key); err != nil {
				return err
			} else {
				fmt.Println("delete successfully")
			}
			return nil
		},
	}
	hasCmd = &cobra.Command{
		Use:   "has [key]",
		Short: "Checks if a key exists",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			if found, err := rpcStore.Has(key); err != nil {
				return err
			} else {
				fmt.Printf("key=%s, found=%t\n", key, found)
			}
			return nil
		},
	}
)
