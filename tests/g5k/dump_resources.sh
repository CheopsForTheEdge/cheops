#!/usr/bin/env sh

# dump_resources.sh will list all resources known to the frontend and group nodes that have the same version of the content. Example:
#
# 123abc
# 	node1
# 	node2
# 		{
#				"_id": "123abc",
# 			"Locations": ["node1", "node2", "node3"],
#			  "Units": [
# 				<first unit>,
# 				<second unit>
# 			]
#			}
#		node3
# 		{
#				"_id": "123abc",
# 			"Locations": ["node1", "node2", "node3"],
#			  "Units": [
# 				<first unit>,
# 				<second unit>,
# 				<third unit>
# 			]
#			}
#
# Here we have one resource with id "123abc", on 3 nodes, the content is the same on node1 and node2 but differs on node3

. ./env.sh


alldocs=$(mktemp)

for node in $(cat ~/.oarnodes)
do
				bookmark=""
				f=$(mktemp)
				g=$(mktemp)
				while true
				do
								curl -s -XPOST -H 'Content-Type: application/json' $node:5984/cheops/_find --data-binary "{\"selector\": {\"Type\": \"RESOURCE\"}, \"fields\": [\"_id\"], \"bookmark\": \"$bookmark\"}" > $f

								jq -r '.docs[] | ._id' $f | tee $g
								if [ "$(stat -c '%s' $g)" = "0" ]; then
												break
							 	fi
								bookmark=$(jq -r '.bookmark' $f)
				done
				rm $f
				rm $g
done | sort | uniq > $alldocs

dir=$(mktemp -d)
content=$(mktemp)
cat $alldocs | while read id
do
				for node in $(cat ~/.oarnodes)
				do
								curl -s $node:5984/cheops/$id | jq '.' > $content
								cat $content | jq '.error' | grep -q "not_found" && continue
								sum=$(cat $content | md5sum | awk '{print $1}')
								mkdir -p $dir/$id/$sum
								echo $node >> $dir/$id/$sum/nodes
								cat $content > $dir/$id/$sum/content
				done
done
rm $content
rm $alldocs

ls $dir | while read id
do
				echo $id
				ls $dir/$id | while read sum
				do
								cat $dir/$id/$sum/nodes | awk '{print "\t" $0}'
								cat $dir/$id/$sum/content | awk '{print "\t\t" $0}'
				done
done

rm -r $dir
