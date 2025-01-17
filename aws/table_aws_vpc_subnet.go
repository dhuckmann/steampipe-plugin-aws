package aws

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/turbot/steampipe-plugin-sdk/v3/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v3/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v3/plugin/transform"
)

func tableAwsVpcSubnet(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "aws_vpc_subnet",
		Description: "AWS VPC Subnet",
		Get: &plugin.GetConfig{
			KeyColumns:        plugin.SingleColumn("subnet_id"),
			ShouldIgnoreError: isNotFoundError([]string{"InvalidSubnetID.Malformed", "InvalidSubnetID.NotFound"}),
			Hydrate:           getVpcSubnet,
		},
		List: &plugin.ListConfig{
			Hydrate: listVpcSubnets,
			KeyColumns: []*plugin.KeyColumn{
				{Name: "availability_zone", Require: plugin.Optional},
				{Name: "availability_zone_id", Require: plugin.Optional},
				{Name: "available_ip_address_count", Require: plugin.Optional},
				{Name: "cidr_block", Require: plugin.Optional},
				{Name: "default_for_az", Require: plugin.Optional},
				{Name: "outpost_arn", Require: plugin.Optional},
				{Name: "owner_id", Require: plugin.Optional},
				{Name: "state", Require: plugin.Optional},
				{Name: "subnet_arn", Require: plugin.Optional},
				{Name: "vpc_id", Require: plugin.Optional},
			},
		},
		GetMatrixItem: BuildRegionList,
		Columns: awsRegionalColumns([]*plugin.Column{
			{
				Name:        "subnet_id",
				Description: "Contains the unique ID to specify a subnet.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "subnet_arn",
				Description: "Contains the Amazon Resource Name (ARN) of the subnet.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "vpc_id",
				Description: "ID of the VPC, the subnet is in.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "cidr_block",
				Description: "Contains the IPv4 CIDR block assigned to the subnet.",
				Type:        proto.ColumnType_CIDR,
			},
			{
				Name:        "state",
				Description: "Current state of the subnet.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "owner_id",
				Description: "Contains the AWS account that own the subnet.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "assign_ipv6_address_on_creation",
				Description: "Indicates whether a network interface created in this subnet (including a network interface created by RunInstances) receives an IPv6 address.",
				Type:        proto.ColumnType_BOOL,
			},
			{
				Name:        "available_ip_address_count",
				Description: "The number of unused private IPv4 addresses in the subnet. The IPv4 addresses for any stopped instances are considered unavailable.",
				Type:        proto.ColumnType_INT,
			},
			{
				Name:        "availability_zone",
				Description: "The Availability Zone of the subnet.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "availability_zone_id",
				Description: "The AZ ID of the subnet.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "customer_owned_ipv4_pool",
				Description: "The customer-owned IPv4 address pool associated with the subnet.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "default_for_az",
				Description: "Indicates whether this is the default subnet for the Availability Zone.",
				Type:        proto.ColumnType_BOOL,
			},
			{
				Name:        "map_customer_owned_ip_on_launch",
				Description: "Indicates whether a network interface created in this subnet (including a network interface created by RunInstances) receives a customer-owned IPv4 address.",
				Type:        proto.ColumnType_BOOL,
			},
			{
				Name:        "map_public_ip_on_launch",
				Description: "Indicates whether instances launched in this subnet receive a public IPv4 address.",
				Type:        proto.ColumnType_BOOL,
			},
			{
				Name:        "outpost_arn",
				Description: "The Amazon Resource Name (ARN) of the Outpost. Available only if subnet is on an outpost.",
				Type:        proto.ColumnType_STRING,
			},
			{
				Name:        "ipv6_cidr_block_association_set",
				Description: "A list of IPv6 CIDR blocks associated with the subnet.",
				Type:        proto.ColumnType_JSON,
			},
			{
				Name:        "tags_src",
				Description: "A list of tags that are attached to the subnet.",
				Type:        proto.ColumnType_JSON,
				Transform:   transform.FromField("Tags"),
			},

			// Standard columns for all tables
			{
				Name:        "tags",
				Description: resourceInterfaceDescription("tags"),
				Type:        proto.ColumnType_JSON,
				Transform:   transform.From(getVpcSubnetTurbotTags),
			},
			{
				Name:        "title",
				Description: resourceInterfaceDescription("title"),
				Type:        proto.ColumnType_STRING,
				Transform:   transform.From(getSubnetTurbotTitle),
			},
			{
				Name:        "akas",
				Description: resourceInterfaceDescription("akas"),
				Type:        proto.ColumnType_JSON,
				Transform:   transform.FromField("SubnetArn").Transform(arnToAkas),
			},
		}),
	}
}

//// LIST FUNCTION

func listVpcSubnets(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (interface{}, error) {
	region := d.KeyColumnQualString(matrixKeyRegion)
	plugin.Logger(ctx).Trace("listVpcSubnets", "AWS_REGION", region)

	// Create session
	svc, err := Ec2Service(ctx, d, region)
	if err != nil {
		return nil, err
	}

	input := &ec2.DescribeSubnetsInput{
		MaxResults: aws.Int64(1000),
	}

	filterKeyMap := []VpcFilterKeyMap{
		{ColumnName: "availability_zone", FilterName: "availability-zone", ColumnType: "string"},
		{ColumnName: "availability_zone_id", FilterName: "availability-zone-id", ColumnType: "string"},
		{ColumnName: "available_ip_address_count", FilterName: "available-ip-address-count", ColumnType: "int64"},
		{ColumnName: "cidr_block", FilterName: "cidr-block", ColumnType: "cidr"},
		{ColumnName: "default_for_az", FilterName: "default-for-az", ColumnType: "boolean"},
		{ColumnName: "outpost_arn", FilterName: "outpost-arn", ColumnType: "string"},
		{ColumnName: "owner_id", FilterName: "owner-id", ColumnType: "string"},
		{ColumnName: "state", FilterName: "state", ColumnType: "string"},
		{ColumnName: "subnet_arn", FilterName: "subnet-arn", ColumnType: "string"},
		{ColumnName: "vpc_id", FilterName: "vpc-id", ColumnType: "string"},
	}

	filters := buildVpcResourcesFilterParameter(filterKeyMap, d.Quals)
	if len(filters) > 0 {
		input.Filters = filters
	}

	// Reduce the basic request limit down if the user has only requested a small number of rows
	limit := d.QueryContext.Limit
	if d.QueryContext.Limit != nil {
		if *limit < *input.MaxResults {
			if *limit < 5 {
				input.MaxResults = aws.Int64(5)
			} else {
				input.MaxResults = limit
			}
		}
	}

	// List call
	err = svc.DescribeSubnetsPages(
		input,
		func(page *ec2.DescribeSubnetsOutput, isLast bool) bool {
			for _, subnet := range page.Subnets {
				d.StreamListItem(ctx, subnet)

				// Context may get cancelled due to manual cancellation or if the limit has been reached
				if d.QueryStatus.RowsRemaining(ctx) == 0 {
					return false
				}
			}
			return !isLast
		},
	)

	return nil, err
}

//// HYDRATE FUNCTIONS

func getVpcSubnet(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (interface{}, error) {
	plugin.Logger(ctx).Trace("getVpcSubnet")

	region := d.KeyColumnQualString(matrixKeyRegion)
	subnetID := d.KeyColumnQuals["subnet_id"].GetStringValue()

	// get service
	svc, err := Ec2Service(ctx, d, region)
	if err != nil {
		return nil, err
	}

	// Build the params
	params := &ec2.DescribeSubnetsInput{
		SubnetIds: []*string{aws.String(subnetID)},
	}

	// Get call
	op, err := svc.DescribeSubnets(params)
	if err != nil {
		plugin.Logger(ctx).Debug("getVpcSubnet__", "ERROR", err)
		return nil, err
	}

	if op.Subnets != nil && len(op.Subnets) > 0 {
		return op.Subnets[0], nil
	}
	return nil, nil
}

//// TRANSFORM FUNCTIONS

func getVpcSubnetTurbotTags(_ context.Context, d *transform.TransformData) (interface{}, error) {
	subnet := d.HydrateItem.(*ec2.Subnet)
	return ec2TagsToMap(subnet.Tags)
}

func getSubnetTurbotTitle(_ context.Context, d *transform.TransformData) (interface{}, error) {
	subnet := d.HydrateItem.(*ec2.Subnet)
	subnetData := d.HydrateResults
	var title string
	if subnet.Tags != nil {
		for _, i := range subnet.Tags {
			if *i.Key == "Name" {
				title = *i.Value
			}
		}
	}

	if title == "" {
		if subnetData["getVpcSubnet"] != nil {
			title = *subnetData["getVpcSubnet"].(*ec2.Subnet).SubnetId
		} else {
			title = *subnetData["listVpcSubnets"].(*ec2.Subnet).SubnetId
		}
	}
	return title, nil
}
